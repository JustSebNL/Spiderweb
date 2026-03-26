#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PATCHES_DIR="${PATCHES_DIR:-$REPO_DIR/infra/vllm/patches}"

BRAIN_DIR="${BRAIN_DIR:-$REPO_DIR/brain}"
BRAIN_MODEL_CACHE_DIR="${BRAIN_MODEL_CACHE_DIR:-${YOUTU_CACHE_DIR:-$BRAIN_DIR/model-cache}}"
HF_HOME="${HF_HOME:-${HF_HOME_DIR:-$HOME/.spiderweb/hf}}"
HF_HUB_CACHE="${HF_HUB_CACHE:-${HF_HUB_CACHE_DIR:-$HF_HOME/hub}}"
BRAIN_VLLM_VENV="${BRAIN_VLLM_VENV:-${YOUTU_VLLM_VENV:-$BRAIN_DIR/.venv-vllm}}"
BRAIN_VLLM_PORT="${BRAIN_VLLM_PORT:-${YOUTU_VLLM_PORT:-8000}}"
BRAIN_VLLM_HOST="${BRAIN_VLLM_HOST:-${YOUTU_VLLM_HOST:-127.0.0.1}}"
BRAIN_VLLM_PID_FILE="${BRAIN_VLLM_PID_FILE:-${YOUTU_VLLM_PID_FILE:-$BRAIN_DIR/brain-vllm.pid}}"
BRAIN_VLLM_LOG_FILE="${BRAIN_VLLM_LOG_FILE:-${YOUTU_VLLM_LOG_FILE:-$BRAIN_DIR/brain-vllm.log}}"
BRAIN_VLLM_MODEL_PATH="${BRAIN_VLLM_MODEL_PATH:-${YOUTU_VLLM_MODEL_PATH:-$BRAIN_DIR}}"
BRAIN_VLLM_MAX_MODEL_LEN="${BRAIN_VLLM_MAX_MODEL_LEN:-${YOUTU_VLLM_MAX_MODEL_LEN:-32768}}"
BRAIN_VLLM_GPU_MEMORY_UTILIZATION="${BRAIN_VLLM_GPU_MEMORY_UTILIZATION:-${YOUTU_VLLM_GPU_MEMORY_UTILIZATION:-0.85}}"
PYTHON_BIN="${BRAIN_VLLM_VENV}/bin/python"

for required_patch in youtu_llm.py configuration_youtu.py registry.py __init__.py; do
  if [ ! -f "$PATCHES_DIR/$required_patch" ]; then
    echo "[ERROR] Native vLLM requires $required_patch under $PATCHES_DIR" >&2
    exit 1
  fi
done

if [ ! -x "$PYTHON_BIN" ]; then
  echo "[ERROR] Missing vLLM virtualenv python at $PYTHON_BIN" >&2
  exit 1
fi

if [ -f "$BRAIN_VLLM_PID_FILE" ] && kill -0 "$(cat "$BRAIN_VLLM_PID_FILE")" 2>/dev/null; then
  echo "[INFO] brain-vllm is already running"
  exit 0
fi

mkdir -p "$BRAIN_DIR" "$BRAIN_MODEL_CACHE_DIR" "$HF_HOME" "$HF_HUB_CACHE"

nohup env HF_HOME="$HF_HOME" HF_HUB_CACHE="$HF_HUB_CACHE" "$PYTHON_BIN" -m vllm.entrypoints.openai.api_server \
  --model "$BRAIN_VLLM_MODEL_PATH" \
  --trust-remote-code \
  --host "$BRAIN_VLLM_HOST" \
  --port "$BRAIN_VLLM_PORT" \
  --max-model-len "$BRAIN_VLLM_MAX_MODEL_LEN" \
  --gpu-memory-utilization "$BRAIN_VLLM_GPU_MEMORY_UTILIZATION" \
  > "$BRAIN_VLLM_LOG_FILE" 2>&1 &

echo $! > "$BRAIN_VLLM_PID_FILE"
echo "[INFO] brain-vllm started with pid $(cat "$BRAIN_VLLM_PID_FILE")"
