# opencode-pre-commit

A Git `commit-msg` hook that validates commit messages against staged diffs using an [opencode](https://opencode.ai) server.

## What it does

- Runs automatically on `git commit`
- Automatically starts opencode server if not running
- Reads the commit message
- Reads the staged diff
- Sends both to LLM for semantic comparison
- **Pass**: commit proceeds
- **Fail**: commit is blocked, shows suggested message for agent
- **Warn**: by default proceeds (configurable)

## Install

```bash
curl -sSL https://raw.githubusercontent.com/0FL01/opencode-pre-commit/main/install.sh | bash
```

This installs the hook at `.git/hooks/commit-msg` and downloads the binary to `.git/hooks/opencode-pre-commit`.

## Configuration (optional)

Create `.opencode-pre-commit.json` in your repo root to override defaults:

```json
{
  "base_url": "http://127.0.0.1:4096",
  "timeout": "5m",
  "fail_statuses": ["fail"],
  "model": "zai-coding-plan/glm-4.7"
}
```

### Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `base_url` | string | `http://127.0.0.1:4096` | opencode server URL |
| `timeout` | string | `5m` | request timeout |
| `fail_statuses` | []string | `["fail"]` | statuses that block commit |
| `model` | string | (none) | force specific LLM model |
| `prompt` | string | (see below) | custom prompt instruction |

### Default prompt

```
Evaluate whether the commit message accurately and sufficiently describes the staged changes.
Return pass if it is correct, warn if it is broadly correct but too vague,
and fail if it is misleading or describes a different primary change.
```

## Usage

Normal workflow:

```bash
git add .
git commit -m "fix(auth): normalize token validation"
# hook validates message against diff
# if ok -> commit proceeds
# if bad -> commit blocked, suggested message shown
```

### Example output (pass)

```
Review status: pass
Accuracy: correct
Completeness: sufficient
Summary: The commit message accurately describes the primary change.
```

### Example output (fail)

```
Review status: fail
Accuracy: incorrect
Completeness: insufficient
Summary: The commit message does not match the primary change.

  [error] wrong_scope: The message claims a test-related change, but the diff primarily modifies auth token normalization.
    - evidence: Added TokenNormalizer struct
    - evidence: Changes validateToken() to call normalize()
    suggested: fix(auth): normalize token format before validation

commit message review status "fail" is configured to fail
```

## Uninstall

```bash
rm .git/hooks/commit-msg .git/hooks/opencode-pre-commit
```

## Development

Run tests:

```bash
go test -v ./...
```

Build:

```bash
go build -o opencode-pre-commit .
```
