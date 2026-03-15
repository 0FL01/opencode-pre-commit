package main

import (
	"strings"
	"testing"
)

func TestBuildPrompt_ContainsPromptInstruction(t *testing.T) {
	cfg := Config{Prompt: "Check for security issues."}
	result := buildPrompt(cfg, "some diff")
	if !strings.Contains(result, "Check for security issues.") {
		t.Error("prompt should contain the config prompt instruction")
	}
}

func TestBuildPrompt_ContainsStatuses(t *testing.T) {
	cfg := Config{Prompt: "review"}
	result := buildPrompt(cfg, "diff")
	for _, s := range allStatuses {
		if !strings.Contains(result, string(s)) {
			t.Errorf("prompt should contain status %q", s)
		}
	}
}

func TestBuildPrompt_ContainsDiff(t *testing.T) {
	cfg := Config{Prompt: "review"}
	diff := "+added line\n-removed line"
	result := buildPrompt(cfg, diff)
	if !strings.Contains(result, diff) {
		t.Error("prompt should contain the diff")
	}
	if !strings.Contains(result, "```diff") {
		t.Error("prompt should wrap diff in fenced code block")
	}
}

func TestBuildPrompt_RequestsJSONOnly(t *testing.T) {
	cfg := Config{Prompt: "review"}
	result := buildPrompt(cfg, "diff")
	if !strings.Contains(result, "Respond ONLY with a JSON object") {
		t.Error("prompt should request JSON-only response")
	}
}

func TestBuildPrompt_ContainsReviewerRole(t *testing.T) {
	cfg := Config{Prompt: "review"}
	result := buildPrompt(cfg, "diff")
	if !strings.Contains(result, "You are a code reviewer") {
		t.Error("prompt should set reviewer role")
	}
}

func TestBuildPrompt_DefaultPrompt(t *testing.T) {
	cfg := defaultConfig()
	result := buildPrompt(cfg, "diff")
	if !strings.Contains(result, defaultPrompt) {
		t.Error("default config should use default prompt")
	}
}
