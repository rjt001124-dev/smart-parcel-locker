package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestResolveToolBin(t *testing.T) {
	tests := []struct {
		name   string
		goos   string
		gobin  string
		gopath string
		want   string
	}{
		{name: "explicit GOBIN wins", goos: "windows", gobin: `E:\custom tools\bin`, gopath: `D:\first;E:\second`, want: "E:/custom tools/bin"},
		{name: "Windows drive root GOBIN", goos: "windows", gobin: `C:\`, want: "C:/"},
		{name: "Unix root GOBIN", goos: "linux", gobin: "/", want: "/"},
		{name: "first Windows GOPATH entry", goos: "windows", gopath: `D:\first;E:\second`, want: "D:/first/bin"},
		{name: "Windows drive root GOPATH", goos: "windows", gopath: `C:\;D:\second`, want: "C:/bin"},
		{name: "first Unix GOPATH entry", goos: "linux", gopath: "/home/user/first:/srv/second", want: "/home/user/first/bin"},
		{name: "Unix root GOPATH", goos: "linux", gopath: "/:/srv/second", want: "/bin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveToolBin(tt.goos, tt.gobin, tt.gopath)
			if err != nil {
				t.Fatalf("resolveToolBin() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveToolBin() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadGoEnvironmentUsesHostOSWhenTargetDiffers(t *testing.T) {
	targetOS := "linux"
	if runtime.GOOS == targetOS {
		targetOS = "windows"
	}
	t.Setenv("GOOS", targetOS)
	t.Setenv("GOARCH", "arm64")

	environment, err := readGoEnvironment()
	if err != nil {
		t.Fatalf("readGoEnvironment() error = %v", err)
	}
	if environment.GOHOSTOS != runtime.GOOS {
		t.Fatalf("GOHOSTOS = %q, want host %q while target GOOS is %q", environment.GOHOSTOS, runtime.GOOS, targetOS)
	}
}

func TestResolveToolBinRejectsMissingGoPaths(t *testing.T) {
	if _, err := resolveToolBin("windows", "", ""); err == nil {
		t.Fatal("resolveToolBin() error = nil, want missing path error")
	}
}

func TestMakefileUsesShellNeutralToolEnvironment(t *testing.T) {
	makefilePath := filepath.Join("..", "..", "Makefile")
	contents, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	makefile := string(contents)

	inlineAssignment := regexp.MustCompile(`(?m)^\t(?:GOBIN|PATH)=[^\r\n]*`)
	if match := inlineAssignment.FindString(makefile); match != "" {
		t.Fatalf("Makefile contains shell-specific inline environment assignment: %q", match)
	}
	for _, required := range []string{
		"unexport GOOS",
		"unexport GOARCH",
		"GOHOSTOS := $(strip $(shell go env GOHOSTOS))",
		"GO_EXE := $(if $(filter windows,$(GOHOSTOS)),.exe,)",
		"PATH_SEPARATOR := $(if $(filter windows,$(GOHOSTOS)),;,:)",
		"export GOBIN := $(GO_TOOL_BIN)",
		"export PATH := $(GO_TOOL_BIN)$(PATH_SEPARATOR)$(PATH)",
		"\t\"$(BUF)\" generate",
	} {
		if !strings.Contains(makefile, required) {
			t.Fatalf("Makefile missing shell-neutral formulation %q", required)
		}
	}
	if strings.Contains(makefile, "GO_EXE := $(if $(filter windows,$(GOOS))") || strings.Contains(makefile, "PATH_SEPARATOR := $(if $(filter windows,$(GOOS))") {
		t.Fatal("Makefile derives host tool behavior from target GOOS")
	}
	toolResolution := strings.Index(makefile, "GO_TOOL_BIN :=")
	if toolResolution < 0 || strings.Index(makefile, "unexport GOOS") > toolResolution || strings.Index(makefile, "unexport GOARCH") > toolResolution {
		t.Fatal("Makefile must unexport target GOOS and GOARCH before running the host tool resolver")
	}
}
