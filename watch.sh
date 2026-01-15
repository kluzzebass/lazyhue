#!/usr/bin/env bash
set -euo pipefail

# Build and watch for changes, restarting the TUI on rebuild
# Requires: fswatch (brew install fswatch)

bin="build/lazyhue"

build() {
  go build -o "$bin" ./cmd/lazyhue
}

pid=

start() {
  "$bin" &
  pid=$!
}

stop() {
  if [ -n "${pid:-}" ] && kill -0 "$pid" 2>/dev/null; then
    kill "$pid"
    wait "$pid" 2>/dev/null || true
  fi
}

trap stop EXIT

# Initial build and start
build
start

# Watch for Go file changes and rebuild
fswatch -o -l 0.5 --include='\.go$' --exclude='.*' . | while read -r _; do
  echo "Change detected, rebuilding..."
  stop
  if build; then
    echo "Build succeeded, restarting..."
    start
  else
    echo "Build failed"
  fi
done
