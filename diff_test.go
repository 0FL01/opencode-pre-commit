package main

import (
	"errors"
	"strings"
	"testing"
)

func TestStagedDiff_Success(t *testing.T) {
	fake := func(name string, args ...string) ([]byte, error) {
		if name != "git" {
			t.Fatalf("expected git command, got %q", name)
		}
		return []byte("diff --git a/main.go b/main.go\n"), nil
	}
	diff, err := stagedDiff(fake)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != "diff --git a/main.go b/main.go\n" {
		t.Errorf("unexpected diff: %q", diff)
	}
}

func TestStagedDiff_Error(t *testing.T) {
	fake := func(name string, args ...string) ([]byte, error) {
		return nil, errors.New("command failed")
	}
	_, err := stagedDiff(fake)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "git diff: command failed" {
		t.Errorf("error = %q, want %q", got, "git diff: command failed")
	}
}

func TestStagedDiff_EmptyOutput(t *testing.T) {
	fake := func(name string, args ...string) ([]byte, error) {
		return []byte(""), nil
	}
	diff, err := stagedDiff(fake)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty diff, got %q", diff)
	}
}

func TestDefaultExecOutput(t *testing.T) {
	// Run a simple command to verify the wrapper works.
	out, err := defaultExecOutput("echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestDefaultExecOutput_Error(t *testing.T) {
	_, err := defaultExecOutput("nonexistent-command-xyz")
	if err == nil {
		t.Fatal("expected error for nonexistent command")
	}
}

func TestStagedDiff_Arguments(t *testing.T) {
	var gotArgs []string
	fake := func(name string, args ...string) ([]byte, error) {
		gotArgs = append([]string{name}, args...)
		return []byte("ok"), nil
	}
	_, _ = stagedDiff(fake)
	expected := []string{"git", "diff", "--cached", "--diff-algorithm=minimal"}
	if len(gotArgs) != len(expected) {
		t.Fatalf("args = %v, want %v", gotArgs, expected)
	}
	for i, arg := range expected {
		if gotArgs[i] != arg {
			t.Errorf("arg[%d] = %q, want %q", i, gotArgs[i], arg)
		}
	}
}
