package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeReviewClient is a test double for ReviewClient.
type fakeReviewClient struct {
	newSessionID  string
	newSessionErr error
	promptText    string
	promptErr     error
	deleteErr     error
	deleteCalled  bool
}

func (f *fakeReviewClient) NewSession(_ context.Context, _ string) (string, error) {
	return f.newSessionID, f.newSessionErr
}

func (f *fakeReviewClient) Prompt(_ context.Context, _ string, _ string) (string, error) {
	return f.promptText, f.promptErr
}

func (f *fakeReviewClient) DeleteSession(_ context.Context, _ string) error {
	f.deleteCalled = true
	return f.deleteErr
}

func TestStripJSONFences_RawJSON(t *testing.T) {
	input := `{"status":"pass","issues":[]}`
	if got := stripJSONFences(input); got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestStripJSONFences_JSONFence(t *testing.T) {
	input := "```json\n{\"status\":\"pass\"}\n```"
	want := `{"status":"pass"}`
	if got := stripJSONFences(input); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStripJSONFences_PlainFence(t *testing.T) {
	input := "```\n{\"status\":\"pass\"}\n```"
	want := `{"status":"pass"}`
	if got := stripJSONFences(input); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStripJSONFences_Whitespace(t *testing.T) {
	input := "  \n ```json\n{\"status\":\"pass\"}\n```  \n"
	want := `{"status":"pass"}`
	if got := stripJSONFences(input); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrintReview_WithIssues(t *testing.T) {
	var buf bytes.Buffer
	r := &Review{
		Status: StatusFail,
		Issues: []Issue{
			{File: "main.go", Line: 10, Severity: "error", Message: "bug found"},
			{File: "util.go", Line: 5, Severity: "warning", Message: "unused var"},
		},
	}
	printReview(&buf, r)
	output := buf.String()
	if !strings.Contains(output, "Review status: fail") {
		t.Errorf("should contain status, got %q", output)
	}
	if !strings.Contains(output, "[error] main.go:10") {
		t.Errorf("should contain first issue, got %q", output)
	}
	if !strings.Contains(output, "[warning] util.go:5") {
		t.Errorf("should contain second issue, got %q", output)
	}
}

func TestPrintReview_NoIssues(t *testing.T) {
	var buf bytes.Buffer
	r := &Review{Status: StatusPass, Issues: []Issue{}}
	printReview(&buf, r)
	output := buf.String()
	if !strings.Contains(output, "No issues found.") {
		t.Errorf("should contain 'No issues found.', got %q", output)
	}
}

func TestPrintReview_NilIssues(t *testing.T) {
	var buf bytes.Buffer
	r := &Review{Status: StatusPass}
	printReview(&buf, r)
	output := buf.String()
	if !strings.Contains(output, "No issues found.") {
		t.Errorf("should contain 'No issues found.', got %q", output)
	}
}

func TestCallOpencode_Success(t *testing.T) {
	client := &fakeReviewClient{
		newSessionID: "sess-1",
		promptText:   `{"status":"pass","issues":[]}`,
	}
	cfg := Config{Timeout: "1m"}
	review, err := callOpencode(context.Background(), client, cfg, "prompt", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.Status != StatusPass {
		t.Errorf("Status = %q, want %q", review.Status, StatusPass)
	}
	if !client.deleteCalled {
		t.Error("DeleteSession should be called")
	}
}

func TestCallOpencode_WithFences(t *testing.T) {
	client := &fakeReviewClient{
		newSessionID: "sess-1",
		promptText:   "```json\n{\"status\":\"warn\",\"issues\":[]}\n```",
	}
	cfg := Config{Timeout: "1m"}
	review, err := callOpencode(context.Background(), client, cfg, "prompt", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.Status != StatusWarn {
		t.Errorf("Status = %q, want %q", review.Status, StatusWarn)
	}
}

func TestCallOpencode_NewSessionError(t *testing.T) {
	client := &fakeReviewClient{
		newSessionErr: errors.New("connection refused"),
	}
	cfg := Config{Timeout: "1m"}
	_, err := callOpencode(context.Background(), client, cfg, "prompt", &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create session") {
		t.Errorf("error = %q, should mention 'create session'", err.Error())
	}
}

func TestCallOpencode_PromptError(t *testing.T) {
	client := &fakeReviewClient{
		newSessionID: "sess-1",
		promptErr:    errors.New("timeout"),
	}
	cfg := Config{Timeout: "1m"}
	_, err := callOpencode(context.Background(), client, cfg, "prompt", &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "prompt") {
		t.Errorf("error = %q, should mention 'prompt'", err.Error())
	}
	if !client.deleteCalled {
		t.Error("DeleteSession should still be called on prompt error")
	}
}

func TestCallOpencode_InvalidJSON(t *testing.T) {
	client := &fakeReviewClient{
		newSessionID: "sess-1",
		promptText:   "not json at all",
	}
	cfg := Config{Timeout: "1m"}
	_, err := callOpencode(context.Background(), client, cfg, "prompt", &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse review JSON") {
		t.Errorf("error = %q, should mention 'parse review JSON'", err.Error())
	}
	if !strings.Contains(err.Error(), "not json at all") {
		t.Errorf("error = %q, should contain raw response", err.Error())
	}
}

func TestCallOpencode_InvalidTimeout(t *testing.T) {
	client := &fakeReviewClient{
		newSessionID: "sess-1",
		promptText:   `{"status":"pass","issues":[]}`,
	}
	cfg := Config{Timeout: "not-a-duration"}
	review, err := callOpencode(context.Background(), client, cfg, "prompt", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.Status != StatusPass {
		t.Errorf("Status = %q, want %q", review.Status, StatusPass)
	}
}

func TestCallOpencode_DeleteWarning(t *testing.T) {
	var warn bytes.Buffer
	client := &fakeReviewClient{
		newSessionID: "sess-1",
		promptText:   `{"status":"pass","issues":[]}`,
		deleteErr:    errors.New("delete failed"),
	}
	cfg := Config{Timeout: "1m"}
	review, err := callOpencode(context.Background(), client, cfg, "prompt", &warn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.Status != StatusPass {
		t.Errorf("Status = %q, want %q", review.Status, StatusPass)
	}
	if !strings.Contains(warn.String(), "warning: failed to delete opencode session") {
		t.Errorf("warn = %q, should contain delete warning", warn.String())
	}
}

func TestCallOpencode_WithIssues(t *testing.T) {
	client := &fakeReviewClient{
		newSessionID: "sess-1",
		promptText:   `{"status":"fail","issues":[{"file":"main.go","line":10,"severity":"error","message":"bug"}]}`,
	}
	cfg := Config{Timeout: "1m"}
	review, err := callOpencode(context.Background(), client, cfg, "prompt", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.Status != StatusFail {
		t.Errorf("Status = %q, want %q", review.Status, StatusFail)
	}
	if len(review.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(review.Issues))
	}
	if review.Issues[0].File != "main.go" {
		t.Errorf("issue file = %q, want %q", review.Issues[0].File, "main.go")
	}
}
