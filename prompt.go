package main

import (
	"fmt"
	"strings"
)

const defaultPrompt = "Look for bugs, security issues, and code style problems."

func buildPrompt(cfg Config, diff string) string {
	statuses := make([]string, len(allStatuses))
	for i, s := range allStatuses {
		statuses[i] = string(s)
	}
	jsonFormat := fmt.Sprintf(`Respond ONLY with a JSON object (no markdown fences, no extra text):
{"status":"%s","issues":[{"file":"...","line":0,"severity":"error|warning|info","message":"..."}]}
If everything looks good, return {"status":"pass","issues":[]}.`, strings.Join(statuses, "|"))

	return "You are a code reviewer. Review the staged git diff below.\n\n" +
		cfg.Prompt + "\n\n" +
		jsonFormat + "\n\n```diff\n" + diff + "\n```"
}
