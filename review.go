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

// Review represents the LLM's assessment of a commit message against a diff.
type Review struct {
	Status       Status  `json:"status"`
	Accuracy     string  `json:"accuracy"`
	Completeness string  `json:"completeness"`
	Summary      string  `json:"summary"`
	Issues       []Issue `json:"issues"`
}

// Issue represents a problem found with the commit message.
type Issue struct {
	Severity         string   `json:"severity"`
	Kind             string   `json:"kind"`
	Message          string   `json:"message"`
	Evidence         []string `json:"evidence,omitempty"`
	SuggestedMessage string   `json:"suggested_message,omitempty"`
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

	sessionID, err := client.NewSession(ctx, "commit-msg review")
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
	fmt.Fprintf(w, "Accuracy: %s\n", r.Accuracy)
	fmt.Fprintf(w, "Completeness: %s\n", r.Completeness)
	fmt.Fprintf(w, "Summary: %s\n", r.Summary)

	if len(r.Issues) > 0 {
		fmt.Fprintln(w, "")
		for _, issue := range r.Issues {
			severity := issue.Severity
			if severity == "" {
				severity = "error"
			}
			fmt.Fprintf(w, "  [%s] %s: %s\n", severity, issue.Kind, issue.Message)

			if len(issue.Evidence) > 0 {
				for _, e := range issue.Evidence {
					fmt.Fprintf(w, "    - evidence: %s\n", e)
				}
			}

			if issue.SuggestedMessage != "" {
				fmt.Fprintf(w, "    suggested: %s\n", issue.SuggestedMessage)
			}
		}
	}
}
