# opencode-pre-commit

A Git pre-commit hook that reviews staged diffs using an [opencode](https://opencode.ai) server.

## Install

In your repo:

```bash
curl -sSL https://raw.githubusercontent.com/plutov/opencode-pre-commit/main/install.sh | bash
```

Make sure to have an opencode server running and accessible at the configured URL.

```bash
opencode serve --port 4096
```

## Configuration

### Server URL

Set the opencode server URL (default: `http://127.0.0.1:4096`) in `.opencode-pre-commit.json`:

```json
{
  "base_url": "http://127.0.0.1:4096",
  "timeout": "5m",
  "fail_statuses": [
    "fail"
  ],
  "prompt": "Check for typos, ensure all variables are used, and verify formatting."
}
```

### Example output

```
go run main.go
Review status: fail
  [warning] main.go:103 — json.Unmarshal error is silently ignored in loadConfig() - malformed config will fail silently
  [warning] main.go:140 — time.ParseDuration error is silently ignored - invalid timeout falls back to default without user notification
  [error] install.sh:17 — go install result is not checked - script continues even if build fails, then tries to use non-existent binary
  [error] opencode-pre-commit:0 — Binary file committed to git - binaries should not be committed to version control
opencode-pre-commit: review status "fail" is configured to fail
exit status 1
```
