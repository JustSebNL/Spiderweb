#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TRIGGER_DIR="${TRIGGER_DIR:-$REPO_DIR/trigger}"
TRIGGER_PID_FILE="${TRIGGER_PID_FILE:-$TRIGGER_DIR/.trigger.pid}"
TRIGGER_LOG_FILE="${TRIGGER_LOG_FILE:-$TRIGGER_DIR/.trigger.log}"
TRIGGER_HOST="${TRIGGER_HOST:-127.0.0.1}"
TRIGGER_PORT="${TRIGGER_PORT:-3030}"

if [ ! -d "$TRIGGER_DIR" ]; then
  echo "[ERROR] Trigger workspace not found at $TRIGGER_DIR" >&2
  exit 1
fi

if [ -f "$TRIGGER_PID_FILE" ] && kill -0 "$(cat "$TRIGGER_PID_FILE")" 2>/dev/null; then
  echo "[INFO] Trigger worker is already running"
  exit 0
fi

mkdir -p "$(dirname "$TRIGGER_PID_FILE")"
cd "$TRIGGER_DIR"

nohup npm run dev -- --hostname "$TRIGGER_HOST" --port "$TRIGGER_PORT" > "$TRIGGER_LOG_FILE" 2>&1 &

echo $! > "$TRIGGER_PID_FILE"
echo "[INFO] Trigger worker started with pid $(cat "$TRIGGER_PID_FILE")"
