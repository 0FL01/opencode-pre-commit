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

func TestPrintReview_Pass(t *testing.T) {
	var buf bytes.Buffer
	r := &Review{
		Status:       StatusPass,
		Accuracy:     "correct",
		Completeness: "sufficient",
		Summary:      "The commit message accurately describes the primary change.",
		Issues:       []Issue{},
	}
	printReview(&buf, r)
	output := buf.String()
	if !strings.Contains(output, "Review status: pass") {
		t.Errorf("should contain status, got %q", output)
	}
	if !strings.Contains(output, "Accuracy: correct") {
		t.Errorf("should contain accuracy, got %q", output)
	}
	if !strings.Contains(output, "Completeness: sufficient") {
		t.Errorf("should contain completeness, got %q", output)
	}
}

func TestPrintReview_WithIssues(t *testing.T) {
	var buf bytes.Buffer
	r := &Review{
		Status:       StatusFail,
		Accuracy:     "incorrect",
		Completeness: "insufficient",
		Summary:      "The commit message does not match the primary change.",
		Issues: []Issue{
			{Severity: "error", Kind: "wrong_scope", Message: "The message claims X but the diff primarily changes Y", Evidence: []string{"evidence 1", "evidence 2"}, SuggestedMessage: "fix: improved message"},
		},
	}
	printReview(&buf, r)
	output := buf.String()
	if !strings.Contains(output, "Review status: fail") {
		t.Errorf("should contain status, got %q", output)
	}
	if !strings.Contains(output, "[error] wrong_scope") {
		t.Errorf("should contain issue with severity and kind, got %q", output)
	}
	if !strings.Contains(output, "evidence: evidence 1") {
		t.Errorf("should contain evidence, got %q", output)
	}
	if !strings.Contains(output, "suggested: fix: improved message") {
		t.Errorf("should contain suggested message, got %q", output)
	}
}

func TestPrintReview_NilIssues(t *testing.T) {
	var buf bytes.Buffer
	r := &Review{Status: StatusPass, Accuracy: "correct", Completeness: "sufficient", Summary: "ok"}
	printReview(&buf, r)
	output := buf.String()
	if !strings.Contains(output, "Summary: ok") {
		t.Errorf("should contain summary, got %q", output)
	}
}

func TestCallOpencode_Success(t *testing.T) {
	client := &fakeReviewClient{
		newSessionID: "sess-1",
		promptText:   `{"status":"pass","accuracy":"correct","completeness":"sufficient","summary":"ok","issues":[]}`,
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
		promptText:   "```json\n{\"status\":\"warn\",\"accuracy\":\"correct\",\"completeness\":\"insufficient\",\"summary\":\"ok\",\"issues\":[]}\n```",
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
		promptText:   `{"status":"pass","accuracy":"correct","completeness":"sufficient","summary":"ok","issues":[]}`,
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
		promptText:   `{"status":"pass","accuracy":"correct","completeness":"sufficient","summary":"ok","issues":[]}`,
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
		promptText:   `{"status":"fail","accuracy":"incorrect","completeness":"insufficient","summary":"bad","issues":[{"severity":"error","kind":"wrong_scope","message":"wrong scope","evidence":["evidence1"],"suggested_message":"fix: better"}]}`,
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
	if review.Issues[0].Kind != "wrong_scope" {
		t.Errorf("issue kind = %q, want %q", review.Issues[0].Kind, "wrong_scope")
	}
	if review.Issues[0].SuggestedMessage != "fix: better" {
		t.Errorf("suggested message = %q, want %q", review.Issues[0].SuggestedMessage, "fix: better")
	}
}
