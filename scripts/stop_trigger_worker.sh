#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TRIGGER_DIR="${TRIGGER_DIR:-$REPO_DIR/trigger}"
TRIGGER_PID_FILE="${TRIGGER_PID_FILE:-$TRIGGER_DIR/.trigger.pid}"

if [ ! -f "$TRIGGER_PID_FILE" ]; then
  echo "[INFO] Trigger worker pid file not found"
  exit 0
fi

PID="$(cat "$TRIGGER_PID_FILE")"
if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
  kill "$PID"
  for _ in $(seq 1 20); do
    if ! kill -0 "$PID" 2>/dev/null; then
      break
    fi
    sleep 0.5
  done
  if kill -0 "$PID" 2>/dev/null; then
    kill -9 "$PID" 2>/dev/null || true
  fi
fi

rm -f "$TRIGGER_PID_FILE"
echo "[INFO] Trigger worker stopped"
