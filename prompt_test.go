package main

import (
	"strings"
	"testing"
)

func TestBuildPrompt_ContainsCommitMessage(t *testing.T) {
	cfg := Config{Prompt: "review"}
	commitMsg := "fix(auth): resolve token validation bug"
	result := buildPrompt(cfg, commitMsg, "some diff")
	if !strings.Contains(result, commitMsg) {
		t.Error("prompt should contain the commit message")
	}
}

func TestBuildPrompt_ContainsDiff(t *testing.T) {
	cfg := Config{Prompt: "review"}
	diff := "+added line\n-removed line"
	result := buildPrompt(cfg, "fix: something", diff)
	if !strings.Contains(result, diff) {
		t.Error("prompt should contain the diff")
	}
	if !strings.Contains(result, "```diff") {
		t.Error("prompt should wrap diff in fenced code block")
	}
}

func TestBuildPrompt_ContainsStatuses(t *testing.T) {
	cfg := Config{Prompt: "review"}
	result := buildPrompt(cfg, "fix: something", "diff")
	for _, s := range allStatuses {
		if !strings.Contains(result, string(s)) {
			t.Errorf("prompt should contain status %q", s)
		}
	}
}

func TestBuildPrompt_RequestsJSONOnly(t *testing.T) {
	cfg := Config{Prompt: "review"}
	result := buildPrompt(cfg, "fix: something", "diff")
	if !strings.Contains(result, "Respond ONLY with a JSON object") {
		t.Error("prompt should request JSON-only response")
	}
}

func TestBuildPrompt_RequiresSuggestedMessage(t *testing.T) {
	cfg := Config{Prompt: "review"}
	result := buildPrompt(cfg, "fix: something", "diff")
	if !strings.Contains(result, "suggested_message") {
		t.Error("prompt should request suggested_message field in issues")
	}
}

func TestBuildPrompt_DefaultPrompt(t *testing.T) {
	cfg := defaultConfig()
	result := buildPrompt(cfg, "fix: something", "diff")
	if !strings.Contains(result, defaultPrompt) {
		t.Error("default config should use default prompt")
	}
}
