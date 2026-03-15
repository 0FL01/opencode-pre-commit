package main

import (
	"io/fs"
	"os"
	"testing"
)

func TestLoadConfig_MissingFile(t *testing.T) {
	cfg, err := loadConfig("nonexistent.json", os.ReadFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := defaultConfig()
	if cfg.BaseURL != want.BaseURL || cfg.Timeout != want.Timeout || cfg.Prompt != want.Prompt {
		t.Fatalf("got %+v, want %+v", cfg, want)
	}
	if len(cfg.FailStatuses) != len(want.FailStatuses) || cfg.FailStatuses[0] != want.FailStatuses[0] {
		t.Fatalf("FailStatuses got %v, want %v", cfg.FailStatuses, want.FailStatuses)
	}
}

func TestLoadConfig_ReadError(t *testing.T) {
	readFile := func(string) ([]byte, error) {
		return nil, fs.ErrPermission
	}
	_, err := loadConfig("config.json", readFile)
	if err == nil {
		t.Fatal("expected error for permission denied")
	}
}

func TestLoadConfig_ValidJSON(t *testing.T) {
	readFile := func(string) ([]byte, error) {
		return []byte(`{"base_url":"http://example.com","timeout":"10m","fail_statuses":["warn"],"prompt":"custom"}`), nil
	}
	cfg, err := loadConfig("config.json", readFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "http://example.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://example.com")
	}
	if cfg.Timeout != "10m" {
		t.Errorf("Timeout = %q, want %q", cfg.Timeout, "10m")
	}
	if len(cfg.FailStatuses) != 1 || cfg.FailStatuses[0] != StatusWarn {
		t.Errorf("FailStatuses = %v, want [warn]", cfg.FailStatuses)
	}
	if cfg.Prompt != "custom" {
		t.Errorf("Prompt = %q, want %q", cfg.Prompt, "custom")
	}
}

func TestLoadConfig_PartialJSON(t *testing.T) {
	readFile := func(string) ([]byte, error) {
		return []byte(`{"base_url":"http://custom.com"}`), nil
	}
	cfg, err := loadConfig("config.json", readFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "http://custom.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://custom.com")
	}
	// Unspecified fields should keep defaults.
	want := defaultConfig()
	if cfg.Timeout != want.Timeout {
		t.Errorf("Timeout = %q, want default %q", cfg.Timeout, want.Timeout)
	}
	if cfg.Prompt != want.Prompt {
		t.Errorf("Prompt = %q, want default %q", cfg.Prompt, want.Prompt)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	readFile := func(string) ([]byte, error) {
		return []byte(`{invalid`), nil
	}
	_, err := loadConfig("config.json", readFile)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.BaseURL != defaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, defaultBaseURL)
	}
	if cfg.Prompt != defaultPrompt {
		t.Errorf("Prompt = %q, want %q", cfg.Prompt, defaultPrompt)
	}
	if len(cfg.FailStatuses) != 1 || cfg.FailStatuses[0] != StatusFail {
		t.Errorf("FailStatuses = %v, want [fail]", cfg.FailStatuses)
	}
}
