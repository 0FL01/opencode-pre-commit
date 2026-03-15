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

	configPath      string
	readFile        func(string) ([]byte, error)
	execOutput      ExecOutput
	newReviewClient func(baseURL string) ReviewClient
	startSpinner    func(w io.Writer, msg string) func()
}

func defaultDeps() deps {
	return deps{
		stdout:          os.Stdout,
		stderr:          os.Stderr,
		configPath:      defaultConfigFile,
		readFile:        os.ReadFile,
		execOutput:      defaultExecOutput,
		newReviewClient: newOpencodeClient,
		startSpinner:    startSpinner,
	}
}

func main() {
	if err := run(context.Background(), defaultDeps()); err != nil {
		fmt.Fprintf(os.Stderr, "opencode-pre-commit: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, d deps) error {
	diff, err := stagedDiff(d.execOutput)
	if err != nil {
		return err
	}
	if strings.TrimSpace(diff) == "" {
		fmt.Fprintln(d.stdout, "No staged changes to review.")
		return nil
	}

	cfg, err := loadConfig(d.configPath, d.readFile)
	if err != nil {
		return err
	}

	prompt := buildPrompt(cfg, diff)

	stop := d.startSpinner(d.stderr, "Reviewing staged changes...")
	review, err := callOpencode(ctx, d.newReviewClient(cfg.BaseURL), cfg, prompt, d.stderr)
	stop()
	if err != nil {
		return err
	}

	printReview(d.stdout, review)

	if slices.Contains(cfg.FailStatuses, review.Status) {
		return fmt.Errorf("review status %q is configured to fail", review.Status)
	}
	return nil
}
