package main

import (
	"fmt"
	"strings"
)

const defaultPrompt = "Evaluate whether the commit message accurately and sufficiently describes the staged changes. Return pass if it is correct, warn if it is broadly correct but too vague, and fail if it is misleading or describes a different primary change."

func buildPrompt(cfg Config, commitMsg, diff string) string {
	statuses := make([]string, len(allStatuses))
	for i, s := range allStatuses {
		statuses[i] = string(s)
	}

	jsonFormat := fmt.Sprintf(`Respond ONLY with a JSON object (no markdown fences, no extra text):
{"status":"%s","accuracy":"correct|partially_correct|incorrect|unclear","completeness":"sufficient|insufficient","summary":"brief explanation","issues":[{"severity":"warning|error","kind":"false_claim|wrong_scope|too_vague|missing_primary_change|unsupported_detail","message":"human readable explanation","evidence":["fact from diff","..."],"suggested_message":"optional better commit message"}]}
If the message is good, return {"status":"pass","accuracy":"correct","completeness":"sufficient","summary":"The commit message accurately describes the primary change.","issues":[]}.`, strings.Join(statuses, "|"))

	return "You are a commit message reviewer. Evaluate whether the commit message accurately describes the staged git diff.\n\n" +
		cfg.Prompt + "\n\n" +
		jsonFormat + "\n\n" +
		"COMMIT MESSAGE:\n" + commitMsg + "\n\n" +
		"STAGED DIFF:\n```diff\n" + diff + "\n```\n"
}
