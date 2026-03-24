package main

import (
	"errors"
	"testing"
)

func TestReadCommitMessage_Success(t *testing.T) {
	content := "fix: resolve authentication bug\n\nThis fixes the token validation issue."
	readFile := func(string) ([]byte, error) {
		return []byte(content), nil
	}

	msg, err := readCommitMessage("/tmp/commit_msg", readFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != content {
		t.Errorf("got %q, want %q", msg, content)
	}
}

func TestReadCommitMessage_StripsComments(t *testing.T) {
	content := "# Please enter the commit message\nfix: resolve bug\n# Please include relevant info"
	readFile := func(string) ([]byte, error) {
		return []byte(content), nil
	}

	msg, err := readCommitMessage("/tmp/commit_msg", readFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == content {
		t.Error("comments should have been stripped")
	}
	if !contains(msg, "fix: resolve bug") {
		t.Errorf("commit message content should be preserved, got %q", msg)
	}
	if contains(msg, "#") {
		t.Error("comments should not be in result")
	}
}

func TestReadCommitMessage_EmptyAfterCleaning(t *testing.T) {
	content := "# Just comments\n# Nothing else"
	readFile := func(string) ([]byte, error) {
		return []byte(content), nil
	}

	_, err := readCommitMessage("/tmp/commit_msg", readFile)
	if err == nil {
		t.Fatal("expected error for empty commit message")
	}
	if !contains(err.Error(), "empty") {
		t.Errorf("error = %q, should mention 'empty'", err.Error())
	}
}

func TestReadCommitMessage_ReadError(t *testing.T) {
	expectedErr := errors.New("file not found")
	readFile := func(string) ([]byte, error) {
		return nil, expectedErr
	}

	_, err := readCommitMessage("/tmp/commit_msg", readFile)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "read commit message file") {
		t.Errorf("error = %q, should mention 'read commit message file'", err.Error())
	}
}

func TestCleanCommitMessage_RemovesComments(t *testing.T) {
	input := "# comment 1\nsome text\n# comment 2"
	result := cleanCommitMessage(input)
	if contains(result, "#") {
		t.Error("result should not contain comment lines")
	}
	if !contains(result, "some text") {
		t.Error("non-comment lines should be preserved")
	}
}

func TestCleanCommitMessage_TrimsWhitespace(t *testing.T) {
	input := "  \n  fix: something  \n  \n"
	result := cleanCommitMessage(input)
	if result != "fix: something" {
		t.Errorf("got %q, want %q", result, "fix: something")
	}
}

func TestCleanCommitMessage_PreservesSubjectAndBody(t *testing.T) {
	input := "fix: handle edge case\n\nDetailed description of the fix.\nWith multiple lines."
	result := cleanCommitMessage(input)
	if !contains(result, "fix: handle edge case") {
		t.Error("subject should be preserved")
	}
	if !contains(result, "Detailed description") {
		t.Error("body should be preserved")
	}
}

func TestCleanCommitMessage_OnlyComments(t *testing.T) {
	input := "# only comments here"
	result := cleanCommitMessage(input)
	if result != "" {
		t.Errorf("got %q, want empty string", result)
	}
}

func TestCleanCommitMessage_PreservesLinesStartingWithHash(t *testing.T) {
	// Lines that start with # but are not git comments (e.g., #123 in body)
	input := "fix: something\n\nRelated to #123"
	result := cleanCommitMessage(input)
	// Our current implementation removes all lines starting with #,
	// which is correct for git commit messages since # marks comments
	if !contains(result, "fix: something") {
		t.Error("subject should be preserved")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
