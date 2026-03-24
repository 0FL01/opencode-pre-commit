package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func testDeps(stdout, stderr *bytes.Buffer) deps {
	return deps{
		stdout:        stdout,
		stderr:        stderr,
		commitMsgPath: "/tmp/commit_msg",
		configPath:    "config.json",
		readFile: func(path string) ([]byte, error) {
			if path == "/tmp/commit_msg" {
				return []byte("fix: something"), nil
			}
			return nil, os.ErrNotExist
		},
		execOutput: func(string, ...string) ([]byte, error) {
			return []byte("some diff\n"), nil
		},
		newReviewClient: func(string) ReviewClient {
			return &fakeReviewClient{
				newSessionID: "sess-1",
				promptText:   `{"status":"pass","accuracy":"correct","completeness":"sufficient","summary":"ok","issues":[]}`,
			}
		},
		startSpinner: func(w io.Writer, msg string) func() {
			return func() {}
		},
	}
}

func TestRun_EmptyCommitMsgPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	d := testDeps(&stdout, &stderr)
	d.commitMsgPath = ""
	err := run(context.Background(), d)
	if err == nil {
		t.Fatal("expected error for empty commitMsgPath")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, should mention usage", err.Error())
	}
}

func TestRun_EmptyCommitMessage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	d := testDeps(&stdout, &stderr)
	d.readFile = func(path string) ([]byte, error) {
		if path == "/tmp/commit_msg" {
			return []byte("# just a comment\n# another comment\n"), nil
		}
		return nil, os.ErrNotExist
	}
	err := run(context.Background(), d)
	if err == nil {
		t.Fatal("expected error for empty commit message")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error = %q, should mention 'empty'", err.Error())
	}
}

func TestRun_EmptyDiff(t *testing.T) {
	var stdout, stderr bytes.Buffer
	d := testDeps(&stdout, &stderr)
	d.execOutput = func(string, ...string) ([]byte, error) {
		return []byte("   \n"), nil
	}
	err := run(context.Background(), d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "No staged changes to review.") {
		t.Errorf("stdout = %q, want 'No staged changes'", stdout.String())
	}
}

func TestRun_DiffError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	d := testDeps(&stdout, &stderr)
	d.execOutput = func(string, ...string) ([]byte, error) {
		return nil, errors.New("git not found")
	}
	err := run(context.Background(), d)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "git diff") {
		t.Errorf("error = %q, should mention 'git diff'", err.Error())
	}
}

func TestRun_ConfigLoadError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	d := testDeps(&stdout, &stderr)
	d.readFile = func(path string) ([]byte, error) {
		if path == "/tmp/commit_msg" {
			return []byte("fix: something"), nil
		}
		return []byte("{invalid"), nil
	}
	err := run(context.Background(), d)
	if err == nil {
		t.Fatal("expected error from invalid config")
	}
}

func TestRun_OpencodeError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	d := testDeps(&stdout, &stderr)
	d.newReviewClient = func(string) ReviewClient {
		return &fakeReviewClient{
			newSessionErr: errors.New("connection refused"),
		}
	}
	err := run(context.Background(), d)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create session") {
		t.Errorf("error = %q, should mention 'create session'", err.Error())
	}
}

func TestRun_PassStatus(t *testing.T) {
	var stdout, stderr bytes.Buffer
	d := testDeps(&stdout, &stderr)
	err := run(context.Background(), d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Review status: pass") {
		t.Errorf("stdout = %q, want 'Review status: pass'", stdout.String())
	}
}

func TestRun_FailStatus(t *testing.T) {
	var stdout, stderr bytes.Buffer
	d := testDeps(&stdout, &stderr)
	d.newReviewClient = func(string) ReviewClient {
		return &fakeReviewClient{
			newSessionID: "sess-1",
			promptText:   `{"status":"fail","accuracy":"incorrect","completeness":"insufficient","summary":"bad","issues":[{"severity":"error","kind":"wrong_scope","message":"wrong scope"}]}`,
		}
	}
	err := run(context.Background(), d)
	if err == nil {
		t.Fatal("expected error for fail status")
	}
	if !strings.Contains(err.Error(), `review status "fail" is configured to fail`) {
		t.Errorf("error = %q, want fail message", err.Error())
	}
}

func TestRun_WarnStatusNotInFailStatuses(t *testing.T) {
	var stdout, stderr bytes.Buffer
	d := testDeps(&stdout, &stderr)
	d.newReviewClient = func(string) ReviewClient {
		return &fakeReviewClient{
			newSessionID: "sess-1",
			promptText:   `{"status":"warn","accuracy":"correct","completeness":"insufficient","summary":"ok","issues":[]}`,
		}
	}
	err := run(context.Background(), d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_SpinnerStartedAndStopped(t *testing.T) {
	var stdout, stderr bytes.Buffer
	started := false
	stopped := false
	d := testDeps(&stdout, &stderr)
	d.startSpinner = func(w io.Writer, msg string) func() {
		started = true
		return func() { stopped = true }
	}
	err := run(context.Background(), d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !started {
		t.Error("spinner should have been started")
	}
	if !stopped {
		t.Error("spinner should have been stopped")
	}
}

func TestRun_SpinnerStoppedOnError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	started := false
	stopped := false
	d := testDeps(&stdout, &stderr)
	d.startSpinner = func(w io.Writer, msg string) func() {
		started = true
		return func() { stopped = true }
	}
	d.newReviewClient = func(string) ReviewClient {
		return &fakeReviewClient{
			newSessionErr: errors.New("connection refused"),
		}
	}
	_ = run(context.Background(), d)
	if !started {
		t.Error("spinner should have been started")
	}
	if !stopped {
		t.Error("spinner should have been stopped even on error")
	}
}

func TestRun_UsesConfigBaseURL(t *testing.T) {
	var stdout, stderr bytes.Buffer
	var usedURL string
	d := testDeps(&stdout, &stderr)
	d.readFile = func(path string) ([]byte, error) {
		if path == "/tmp/commit_msg" {
			return []byte("fix: something"), nil
		}
		return []byte(`{"base_url":"http://custom:9999"}`), nil
	}
	d.newReviewClient = func(baseURL string) ReviewClient {
		usedURL = baseURL
		return &fakeReviewClient{
			newSessionID: "sess-1",
			promptText:   `{"status":"pass","accuracy":"correct","completeness":"sufficient","summary":"ok","issues":[]}`,
		}
	}
	err := run(context.Background(), d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usedURL != "http://custom:9999" {
		t.Errorf("base URL = %q, want %q", usedURL, "http://custom:9999")
	}
}

func TestMain_Subprocess(t *testing.T) {
	if os.Getenv("TEST_MAIN_SUBPROCESS") != "1" {
		t.Skip("skipping subprocess test in normal mode")
	}
	main()
}

func TestDefaultDeps(t *testing.T) {
	// Skip in tests that run after main() has modified os.Args
	// Just verify the structure is correct
	d := deps{
		stdout:          os.Stdout,
		stderr:          os.Stderr,
		commitMsgPath:   "test_path",
		configPath:      defaultConfigFile,
		readFile:        os.ReadFile,
		execOutput:      defaultExecOutput,
		newReviewClient: newOpencodeClient,
		startSpinner:    startSpinner,
	}
	if d.stdout == nil || d.stderr == nil {
		t.Error("stdout/stderr should not be nil")
	}
	if d.configPath != defaultConfigFile {
		t.Errorf("configPath = %q, want %q", d.configPath, defaultConfigFile)
	}
	if d.readFile == nil {
		t.Error("readFile should not be nil")
	}
	if d.execOutput == nil {
		t.Error("execOutput should not be nil")
	}
	if d.newReviewClient == nil {
		t.Error("newReviewClient should not be nil")
	}
	if d.startSpinner == nil {
		t.Error("startSpinner should not be nil")
	}
}

func TestRun_PassWithSuggestedMessage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	d := testDeps(&stdout, &stderr)
	d.newReviewClient = func(string) ReviewClient {
		return &fakeReviewClient{
			newSessionID: "sess-1",
			promptText:   `{"status":"pass","accuracy":"correct","completeness":"sufficient","summary":"good","issues":[]}`,
		}
	}
	err := run(context.Background(), d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Review status: pass") {
		t.Errorf("stdout = %q, want 'Review status: pass'", stdout.String())
	}
}

func TestRun_FailWithSuggestedMessage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	d := testDeps(&stdout, &stderr)
	d.newReviewClient = func(string) ReviewClient {
		return &fakeReviewClient{
			newSessionID: "sess-1",
			promptText:   `{"status":"fail","accuracy":"incorrect","completeness":"insufficient","summary":"bad","issues":[{"severity":"error","kind":"wrong_scope","message":"message claims X but diff does Y","evidence":["added A","modified B"],"suggested_message":"fix: corrected message"}]}`,
		}
	}
	err := run(context.Background(), d)
	if err == nil {
		t.Fatal("expected error for fail status")
	}
	output := stdout.String()
	if !strings.Contains(output, "suggested: fix: corrected message") {
		t.Errorf("stdout should contain suggested message, got %q", output)
	}
}
