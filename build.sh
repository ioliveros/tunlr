#!/usr/bin/env bash
set -euo pipefail

# Resolve wails binary
WAILS="${WAILS:-}"
if [ -z "$WAILS" ]; then
  if command -v wails &>/dev/null; then
    WAILS="wails"
  elif [ -x "$HOME/go/bin/wails" ]; then
    WAILS="$HOME/go/bin/wails"
  else
    echo "error: wails not found. Install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest" >&2
    exit 1
  fi
fi

# Version info
VERSION=$(tr -d '[:space:]' < VERSION 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS="-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}"

echo "tunlr ${VERSION} (${COMMIT}) — ${BUILD_DATE}"
echo ""

# Keep the in-app icon in sync with the build source
cp build/appicon.png client/src/assets/appicon.png

"$WAILS" build -ldflags "$LDFLAGS" "$@"

echo ""
echo "→ build/bin/tunlr.app"
