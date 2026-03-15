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
		stdout:     stdout,
		stderr:     stderr,
		configPath: "config.json",
		readFile: func(string) ([]byte, error) {
			return nil, os.ErrNotExist
		},
		execOutput: func(string, ...string) ([]byte, error) {
			return []byte("some diff\n"), nil
		},
		newReviewClient: func(string) ReviewClient {
			return &fakeReviewClient{
				newSessionID: "sess-1",
				promptText:   `{"status":"pass","issues":[]}`,
			}
		},
		startSpinner: func(w io.Writer, msg string) func() {
			return func() {}
		},
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
	d.readFile = func(string) ([]byte, error) {
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
			promptText:   `{"status":"fail","issues":[{"file":"a.go","line":1,"severity":"error","message":"bad"}]}`,
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
			promptText:   `{"status":"warn","issues":[]}`,
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
	d.readFile = func(string) ([]byte, error) {
		return []byte(`{"base_url":"http://custom:9999"}`), nil
	}
	d.newReviewClient = func(baseURL string) ReviewClient {
		usedURL = baseURL
		return &fakeReviewClient{
			newSessionID: "sess-1",
			promptText:   `{"status":"pass","issues":[]}`,
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
	d := defaultDeps()
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
