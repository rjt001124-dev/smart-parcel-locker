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
	GOHOSTOS string
	GOBIN    string
	GOPATH   string
}

func main() {
	environment, err := readGoEnvironment()
	if err == nil {
		var toolBin string
		toolBin, err = resolveToolBin(environment.GOHOSTOS, environment.GOBIN, environment.GOPATH)
		if err == nil {
			fmt.Print(toolBin)
			return
		}
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func readGoEnvironment() (goEnvironment, error) {
	output, err := exec.Command("go", "env", "-json", "GOHOSTOS", "GOBIN", "GOPATH").Output()
	if err != nil {
		return goEnvironment{}, fmt.Errorf("read Go environment: %w", err)
	}
	var environment goEnvironment
	if err := json.Unmarshal(output, &environment); err != nil {
		return goEnvironment{}, fmt.Errorf("decode Go environment: %w", err)
	}
	return environment, nil
}

func resolveToolBin(hostOS, gobin, gopath string) (string, error) {
	if gobin = strings.TrimSpace(gobin); gobin != "" {
		return normalizeToolBin(hostOS, gobin), nil
	}
	separator := ":"
	if strings.EqualFold(strings.TrimSpace(hostOS), "windows") {
		separator = ";"
	}
	entries := strings.Split(strings.TrimSpace(gopath), separator)
	if len(entries) == 0 || strings.TrimSpace(entries[0]) == "" {
		return "", errors.New("go env GOBIN and GOPATH are empty")
	}
	return joinToolBin(normalizeToolBin(hostOS, entries[0])), nil
}

func normalizeToolBin(hostOS, value string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), `\`, "/")
	trimmed := strings.TrimRight(normalized, "/")
	if trimmed == "" && strings.Contains(normalized, "/") {
		return "/"
	}
	if strings.EqualFold(strings.TrimSpace(hostOS), "windows") && len(trimmed) == 2 && trimmed[1] == ':' && strings.HasSuffix(normalized, "/") {
		return trimmed + "/"
	}
	return trimmed
}

func joinToolBin(root string) string {
	if strings.HasSuffix(root, "/") {
		return root + "bin"
	}
	return root + "/bin"
}
