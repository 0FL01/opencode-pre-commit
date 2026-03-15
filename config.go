package main

import (
	"encoding/json"
	"errors"
	"io/fs"
)

const (
	defaultConfigFile = ".opencode-pre-commit.json"
)

type Config struct {
	BaseURL      string   `json:"base_url"`
	Timeout      string   `json:"timeout"`
	FailStatuses []Status `json:"fail_statuses"`
	Prompt       string   `json:"prompt"`
}

func defaultConfig() Config {
	return Config{
		BaseURL:      defaultBaseURL,
		Timeout:      defaultTimeout.String(),
		FailStatuses: []Status{StatusFail},
		Prompt:       defaultPrompt,
	}
}

func loadConfig(path string, readFile func(string) ([]byte, error)) (Config, error) {
	cfg := defaultConfig()

	data, err := readFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
