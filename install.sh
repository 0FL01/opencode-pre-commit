#!/usr/bin/env bash
set -euo pipefail

REPO="0FL01/opencode-pre-commit"
BRANCH="${2:-main}"
HOOK_DIR="$(git rev-parse --show-toplevel)/.git/hooks"
HOOK_PATH="${HOOK_DIR}/commit-msg"
BIN_PATH="${HOOK_DIR}/opencode-pre-commit"
OPENCODE_PORT="${OPENCODE_PORT:-4096}"
OPENCODE_BASE_URL="http://127.0.0.1:${OPENCODE_PORT}"

if [ -f "$HOOK_PATH" ]; then
    echo "A commit-msg hook already exists at $HOOK_PATH"
    echo "Remove it first if you want to replace it."
    exit 1
fi

echo "Downloading opencode-pre-commit..."
if command -v curl &>/dev/null; then
    curl -sL --fail -o "$BIN_PATH" "https://raw.githubusercontent.com/${REPO}/${BRANCH}/opencode-pre-commit" || { echo "Download failed"; exit 1; }
elif command -v wget &>/dev/null; then
    wget -q -O "$BIN_PATH" "https://raw.githubusercontent.com/${REPO}/${BRANCH}/opencode-pre-commit" || { echo "Download failed"; exit 1; }
else
    echo "Error: curl or wget is required"
    exit 1
fi

chmod +x "$BIN_PATH"

cat > "$HOOK_PATH" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail

HOOK_DIR="$(dirname "$0")"
BIN_PATH="${HOOK_DIR}/opencode-pre-commit"
OPENCODE_PORT="${OPENCODE_PORT:-4096}"
OPENCODE_BASE_URL="http://127.0.0.1:${OPENCODE_PORT}"
OPENCODE_PID_FILE="${HOOK_DIR}/.opencode.pid"

# Check if opencode server is already running
is_server_running() {
    curl -s --fail "http://127.0.0.1:${OPENCODE_PORT}/health" > /dev/null 2>&1 || return 1
}

# Start opencode server in background
start_server() {
    echo "Starting opencode server..."
    opencode serve --port "${OPENCODE_PORT}" &
    echo $! > "${OPENCODE_PID_FILE}"
    
    # Wait for server to be ready (max 10 seconds)
    for i in {1..20}; do
        if is_server_running; then
            echo "Opencode server ready"
            return 0
        fi
        sleep 0.5
    done
    
    echo "Warning: opencode server may not be ready"
    return 1
}

# Stop server if we started it
stop_server() {
    if [ -f "${OPENCODE_PID_FILE}" ]; then
        PID=$(cat "${OPENCODE_PID_FILE}")
        if kill -0 "${PID}" 2>/dev/null; then
            kill "${PID}" 2>/dev/null || true
        fi
        rm -f "${OPENCODE_PID_FILE}"
    fi
}

# Cleanup on exit
trap stop_server EXIT

# Check if server needs to be started
NEED_START=false
if ! is_server_running; then
    NEED_START=true
    start_server
fi

# Run the binary with config override for base_url
exec env "OPENCODE_BASE_URL=${OPENCODE_BASE_URL}" "$BIN_PATH" "$@"
EOF

chmod +x "$HOOK_PATH"
echo "Installed commit-msg hook at $HOOK_PATH"
echo "Binary installed at $BIN_PATH"
