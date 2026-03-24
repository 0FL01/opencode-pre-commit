package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"
)

const (
	defaultBaseURL = "http://127.0.0.1:4096"
	defaultTimeout = 5 * time.Minute
)

type deps struct {
	stdout io.Writer
	stderr io.Writer

	commitMsgPath   string
	configPath      string
	readFile        func(string) ([]byte, error)
	execOutput      ExecOutput
	newReviewClient func(baseURL, model string) ReviewClient
}

func defaultDeps() deps {
	commitMsgPath := ""
	if len(os.Args) > 1 {
		commitMsgPath = os.Args[1]
	}
	return deps{
		stdout:          os.Stdout,
		stderr:          os.Stderr,
		commitMsgPath:   commitMsgPath,
		configPath:      defaultConfigFile,
		readFile:        os.ReadFile,
		execOutput:      defaultExecOutput,
		newReviewClient: newOpencodeClientWithModel,
	}
}

func main() {
	if err := run(context.Background(), defaultDeps()); err != nil {
		fmt.Fprintf(os.Stderr, "opencode-pre-commit: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, d deps) error {
	// Validate commit-msg file path argument
	if d.commitMsgPath == "" {
		return fmt.Errorf("usage: opencode-pre-commit <commit-msg-file>")
	}

	// Read and clean commit message
	commitMsg, err := readCommitMessage(d.commitMsgPath, d.readFile)
	if err != nil {
		return err
	}

	// Get staged diff
	diff, err := stagedDiff(d.execOutput)
	if err != nil {
		return err
	}
	if strings.TrimSpace(diff) == "" {
		fmt.Fprintln(d.stdout, "No staged changes to review.")
		return nil
	}

	// Load config
	cfg, err := loadConfig(d.configPath, d.readFile)
	if err != nil {
		return err
	}

	// Build prompt with commit message and diff
	prompt := buildPrompt(cfg, commitMsg, diff)

	// Call LLM
	review, err := callOpencode(ctx, d.newReviewClient(cfg.BaseURL, cfg.Model), cfg, prompt, d.stderr)
	if err != nil {
		return err
	}

	// Print review
	printReview(d.stdout, review)

	// Return error if status is in fail_statuses
	if slices.Contains(cfg.FailStatuses, review.Status) {
		return fmt.Errorf("commit message review status %q is configured to fail", review.Status)
	}
	return nil
}
