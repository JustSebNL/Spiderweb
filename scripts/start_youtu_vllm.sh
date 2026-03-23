#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

YOUTU_DIR="${YOUTU_DIR:-$REPO_DIR/youtu-llm}"
YOUTU_CACHE_DIR="${YOUTU_CACHE_DIR:-$YOUTU_DIR/model-cache}"
YOUTU_VLLM_VENV="${YOUTU_VLLM_VENV:-$YOUTU_DIR/.venv-vllm}"
YOUTU_VLLM_PORT="${YOUTU_VLLM_PORT:-8000}"
YOUTU_VLLM_HOST="${YOUTU_VLLM_HOST:-127.0.0.1}"
YOUTU_VLLM_PID_FILE="${YOUTU_VLLM_PID_FILE:-$YOUTU_DIR/youtu-vllm.pid}"
YOUTU_VLLM_LOG_FILE="${YOUTU_VLLM_LOG_FILE:-$YOUTU_DIR/youtu-vllm.log}"
YOUTU_VLLM_MODEL_PATH="${YOUTU_VLLM_MODEL_PATH:-$YOUTU_DIR}"
YOUTU_VLLM_MAX_MODEL_LEN="${YOUTU_VLLM_MAX_MODEL_LEN:-32768}"
YOUTU_VLLM_GPU_MEMORY_UTILIZATION="${YOUTU_VLLM_GPU_MEMORY_UTILIZATION:-0.85}"
PYTHON_BIN="${YOUTU_VLLM_VENV}/bin/python"

if [ ! -x "$PYTHON_BIN" ]; then
  echo "[ERROR] Missing vLLM virtualenv python at $PYTHON_BIN" >&2
  exit 1
fi

if [ -f "$YOUTU_VLLM_PID_FILE" ] && kill -0 "$(cat "$YOUTU_VLLM_PID_FILE")" 2>/dev/null; then
  echo "[INFO] youtu-vllm is already running"
  exit 0
fi

mkdir -p "$YOUTU_DIR" "$YOUTU_CACHE_DIR"

nohup "$PYTHON_BIN" -m vllm.entrypoints.openai.api_server \
  --model "$YOUTU_VLLM_MODEL_PATH" \
  --trust-remote-code \
  --host "$YOUTU_VLLM_HOST" \
  --port "$YOUTU_VLLM_PORT" \
  --max-model-len "$YOUTU_VLLM_MAX_MODEL_LEN" \
  --gpu-memory-utilization "$YOUTU_VLLM_GPU_MEMORY_UTILIZATION" \
  > "$YOUTU_VLLM_LOG_FILE" 2>&1 &

echo $! > "$YOUTU_VLLM_PID_FILE"
echo "[INFO] youtu-vllm started with pid $(cat "$YOUTU_VLLM_PID_FILE")"
