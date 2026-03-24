#!/usr/bin/env bash
set -euo pipefail

REPO="0FL01/opencode-pre-commit"
HOOK_PATH="$(git rev-parse --show-toplevel)/.git/hooks/commit-msg"

if [ -f "$HOOK_PATH" ]; then
    echo "A commit-msg hook already exists at $HOOK_PATH"
    echo "Remove it first if you want to replace it."
    exit 1
fi

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
esac

# Try to download pre-built binary from GitHub releases
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/opencode-pre-commit-${OS}-${ARCH}"

BIN_DIR="$(mktemp -d)"
BIN_PATH="${BIN_DIR}/opencode-pre-commit"

if command -v curl &>/dev/null; then
    DOWNLOADER=curl
elif command -v wget &>/dev/null; then
    DOWNLOADER=wget
else
    echo "Error: curl or wget is required"
    exit 1
fi

echo "Downloading opencode-pre-commit..."
if [ "$DOWNLOADER" = "curl" ]; then
    if curl -sL --fail -o "$BIN_PATH" "$DOWNLOAD_URL" 2>/dev/null; then
        chmod +x "$BIN_PATH"
        echo "Downloaded to $BIN_PATH"
    fi
else
    if wget -q -O "$BIN_PATH" "$DOWNLOAD_URL" 2>/dev/null; then
        chmod +x "$BIN_PATH"
        echo "Downloaded to $BIN_PATH"
    fi
fi

# Fallback to go install if download failed
if [ ! -f "$BIN_PATH" ] || [ ! -x "$BIN_PATH" ]; then
    echo "Download failed, falling back to go install..."
    go install "github.com/${REPO}@latest"
    BIN_PATH="$(go env GOPATH)/bin/opencode-pre-commit"
fi

cat > "$HOOK_PATH" << EOF
#!/usr/bin/env bash
exec $BIN_PATH "\$1"
EOF

chmod +x "$HOOK_PATH"
echo "Installed commit-msg hook at $HOOK_PATH"
