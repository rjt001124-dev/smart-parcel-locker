package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type goEnvironment struct {
	GOOS   string
	GOBIN  string
	GOPATH string
}

func main() {
	environment, err := readGoEnvironment()
	if err == nil {
		var toolBin string
		toolBin, err = resolveToolBin(environment.GOOS, environment.GOBIN, environment.GOPATH)
		if err == nil {
			fmt.Print(toolBin)
			return
		}
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func readGoEnvironment() (goEnvironment, error) {
	output, err := exec.Command("go", "env", "-json", "GOOS", "GOBIN", "GOPATH").Output()
	if err != nil {
		return goEnvironment{}, fmt.Errorf("read Go environment: %w", err)
	}
	var environment goEnvironment
	if err := json.Unmarshal(output, &environment); err != nil {
		return goEnvironment{}, fmt.Errorf("decode Go environment: %w", err)
	}
	return environment, nil
}

func resolveToolBin(goos, gobin, gopath string) (string, error) {
	if gobin = strings.TrimSpace(gobin); gobin != "" {
		return normalizeToolBin(gobin), nil
	}
	separator := ":"
	if strings.EqualFold(strings.TrimSpace(goos), "windows") {
		separator = ";"
	}
	entries := strings.Split(strings.TrimSpace(gopath), separator)
	if len(entries) == 0 || strings.TrimSpace(entries[0]) == "" {
		return "", errors.New("go env GOBIN and GOPATH are empty")
	}
	return normalizeToolBin(entries[0]) + "/bin", nil
}

func normalizeToolBin(value string) string {
	return strings.TrimRight(strings.ReplaceAll(strings.TrimSpace(value), `\`, "/"), "/")
}
