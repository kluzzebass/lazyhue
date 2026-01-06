#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <binary>"
  exit 1
fi

bin="$1"

if [ ! -x "$bin" ]; then
  echo "error: '$bin' is not executable"
  exit 1
fi

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

# start immediately
start

fswatch -o -l 0.2 "$bin" | while read _; do
  stop
  start
done
