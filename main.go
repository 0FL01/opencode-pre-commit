package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	opencode "github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

const (
	defaultBaseURL = "http://127.0.0.1:4096"
	defaultTimeout = 5 * time.Minute
	configFile     = ".opencode-pre-commit.json"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusFail Status = "fail"
	StatusWarn Status = "warn"
)

var AllStatuses = []Status{StatusPass, StatusFail, StatusWarn}

type Config struct {
	BaseURL      string   `json:"base_url"`
	Timeout      string   `json:"timeout"`
	FailStatuses []Status `json:"fail_statuses"`
	Prompt       string   `json:"prompt"`
}

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

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "opencode-pre-commit: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	diff, err := stagedDiff()
	if err != nil {
		return err
	}
	if strings.TrimSpace(diff) == "" {
		fmt.Println("No staged changes to review.")
		return nil
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	prompt := buildPrompt(cfg, diff)

	stop := startSpinner("Reviewing staged changes...")
	review, err := callOpencode(cfg, prompt)
	stop()
	if err != nil {
		return err
	}

	printReview(review)

	if slices.Contains(cfg.FailStatuses, review.Status) {
		return fmt.Errorf("review status %q is configured to fail", review.Status)
	}
	return nil
}

func stagedDiff() (string, error) {
	out, err := exec.Command("git", "diff", "--cached", "--diff-algorithm=minimal").Output()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return string(out), nil
}

func loadConfig() (Config, error) {
	cfg := Config{
		BaseURL:      defaultBaseURL,
		Timeout:      defaultTimeout.String(),
		FailStatuses: []Status{StatusFail},
		Prompt:       defaultPrompt,
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return cfg, nil
	}

	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

const defaultPrompt = "Look for bugs, security issues, and code style problems."

func buildPrompt(cfg Config, diff string) string {
	reviewInstructions := cfg.Prompt

	statuses := make([]string, len(AllStatuses))
	for i, s := range AllStatuses {
		statuses[i] = string(s)
	}
	jsonFormat := fmt.Sprintf(`Respond ONLY with a JSON object (no markdown fences, no extra text):
{"status":"%s","issues":[{"file":"...","line":0,"severity":"error|warning|info","message":"..."}]}
If everything looks good, return {"status":"pass","issues":[]}.`, strings.Join(statuses, "|"))

	return "You are a code reviewer. Review the staged git diff below.\n\n" +
		reviewInstructions + "\n\n" +
		jsonFormat + "\n\n```diff\n" + diff + "\n```"
}

func callOpencode(cfg Config, prompt string) (*Review, error) {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		timeout = defaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := opencode.NewClient(
		option.WithBaseURL(cfg.BaseURL),
		option.WithMaxRetries(1),
	)

	session, err := client.Session.New(ctx, opencode.SessionNewParams{
		Title: opencode.F("pre-commit review"),
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	resp, err := client.Session.Prompt(ctx, session.ID, opencode.SessionPromptParams{
		Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
			opencode.TextPartInputParam{
				Type: opencode.F(opencode.TextPartInputTypeText),
				Text: opencode.F(prompt),
			},
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("prompt: %w", err)
	}

	defer func() {
		_, deleteErr := client.Session.Delete(context.Background(), session.ID, opencode.SessionDeleteParams{})
		if deleteErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to delete opencode session: %v\n", deleteErr)
		}
	}()

	// Extract text from response parts.
	var text string
	for _, part := range resp.Parts {
		if tp, ok := part.AsUnion().(opencode.TextPart); ok {
			text += tp.Text
		}
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

func startSpinner(msg string) func() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan struct{})
	go func() {
		i := 0
		for {
			select {
			case <-done:
				fmt.Fprintf(os.Stderr, "\r\033[K")
				return
			default:
				fmt.Fprintf(os.Stderr, "\r%s %s", frames[i%len(frames)], msg)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
	return func() { close(done) }
}

func printReview(r *Review) {
	fmt.Printf("Review status: %s\n", r.Status)
	for _, issue := range r.Issues {
		fmt.Printf("  [%s] %s:%d — %s\n", issue.Severity, issue.File, issue.Line, issue.Message)
	}
	if len(r.Issues) == 0 {
		fmt.Println("  No issues found.")
	}
}
