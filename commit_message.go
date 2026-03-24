package main

import (
	"fmt"
	"strings"
)

// readCommitMessage reads and cleans the commit message from the given file path.
// It removes git comment lines (starting with #) and trims whitespace.
func readCommitMessage(path string, readFile func(string) ([]byte, error)) (string, error) {
	data, err := readFile(path)
	if err != nil {
		return "", fmt.Errorf("read commit message file: %w", err)
	}

	content := cleanCommitMessage(string(data))

	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("commit message is empty after removing comments")
	}

	return content, nil
}

// cleanCommitMessage removes git comment lines and normalizes whitespace.
func cleanCommitMessage(content string) string {
	var lines []string

	for _, line := range strings.Split(content, "\n") {
		// Skip lines that are git comments
		if strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}

	// Join lines and trim whitespace
	result := strings.TrimSpace(strings.Join(lines, "\n"))

	// Remove trailing whitespace from each line while preserving structure
	var cleanedLines []string
	for _, line := range strings.Split(result, "\n") {
		cleanedLines = append(cleanedLines, strings.TrimRight(line, " \t"))
	}

	return strings.TrimSpace(strings.Join(cleanedLines, "\n"))
}
