package main

import (
	"fmt"
	"os/exec"
)

type ExecOutput func(name string, args ...string) ([]byte, error)

func defaultExecOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func stagedDiff(execOutput ExecOutput) (string, error) {
	out, err := execOutput("git", "diff", "--cached", "--diff-algorithm=minimal")
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return string(out), nil
}
