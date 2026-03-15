package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusFail Status = "fail"
	StatusWarn Status = "warn"
)

var allStatuses = []Status{StatusPass, StatusFail, StatusWarn}

type Review struct {
	Status Status  `json:"status"`
	Issues []Issue `json:"issues"`
}

type Issue struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type ReviewClient interface {
	NewSession(ctx context.Context, title string) (sessionID string, err error)
	Prompt(ctx context.Context, sessionID string, prompt string) (text string, err error)
	DeleteSession(ctx context.Context, sessionID string) error
}

func callOpencode(ctx context.Context, client ReviewClient, cfg Config, prompt string, warn io.Writer) (*Review, error) {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		timeout = defaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sessionID, err := client.NewSession(ctx, "pre-commit review")
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	defer func() {
		if deleteErr := client.DeleteSession(context.Background(), sessionID); deleteErr != nil {
			fmt.Fprintf(warn, "warning: failed to delete opencode session: %v\n", deleteErr)
		}
	}()

	text, err := client.Prompt(ctx, sessionID, prompt)
	if err != nil {
		return nil, fmt.Errorf("prompt: %w", err)
	}

	text = stripJSONFences(text)

	var review Review
	if err := json.Unmarshal([]byte(text), &review); err != nil {
		return nil, fmt.Errorf("parse review JSON: %w\nraw response:\n%s", err, text)
	}
	return &review, nil
}

func stripJSONFences(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func printReview(w io.Writer, r *Review) {
	fmt.Fprintf(w, "Review status: %s\n", r.Status)
	for _, issue := range r.Issues {
		fmt.Fprintf(w, "  [%s] %s:%d — %s\n", issue.Severity, issue.File, issue.Line, issue.Message)
	}
	if len(r.Issues) == 0 {
		fmt.Fprintln(w, "  No issues found.")
	}
}
