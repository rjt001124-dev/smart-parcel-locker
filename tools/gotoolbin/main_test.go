package main

import (
	"os"
	"path/filepath"
	"regexp"
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
		{name: "first Windows GOPATH entry", goos: "windows", gopath: `D:\first;E:\second`, want: "D:/first/bin"},
		{name: "first Unix GOPATH entry", goos: "linux", gopath: "/home/user/first:/srv/second", want: "/home/user/first/bin"},
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
		"export GOBIN := $(GO_TOOL_BIN)",
		"export PATH := $(GO_TOOL_BIN)$(PATH_SEPARATOR)$(PATH)",
		"\t\"$(BUF)\" generate",
	} {
		if !strings.Contains(makefile, required) {
			t.Fatalf("Makefile missing shell-neutral formulation %q", required)
		}
	}
}
