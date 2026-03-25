#!/usr/bin/env bash
set -euo pipefail

# Spiderweb stack bootstrapper
# Purpose:
# - prepare a bare Linux host
# - install Spiderweb build/runtime prerequisites
# - install Spiderweb core through install_spiderweb.sh
# - optionally prepare Trigger.dev workspace dependencies
# - inspect the machine and autonomously choose cheap-cognition runtime
# - download the matching Youtu model format
# - start the matching local model service
# - emit generated runtime env values
# - leave the system ready to run `sweb wakeup`

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_CONFIG_FILE="${SCRIPT_DIR}/install_spiderweb_stack.conf"
STACK_CONFIG="${STACK_CONFIG:-$DEFAULT_CONFIG_FILE}"

if [ -f "$STACK_CONFIG" ]; then
  # shellcheck source=/dev/null
  . "$STACK_CONFIG"
fi

SPIDERWEB_DIR="${SPIDERWEB_DIR:-$HOME/spiderweb}"
INSTALL_PREFIX="${INSTALL_PREFIX:-$HOME/.local}"
TRIGGER_DIR="${TRIGGER_DIR:-$SPIDERWEB_DIR/trigger}"
BRAIN_DIR="${BRAIN_DIR:-${YOUTU_DIR:-$SPIDERWEB_DIR/brain}}"
YOUTU_MODEL_REPO="${YOUTU_MODEL_REPO:-tencent/Youtu-LLM-2B}"
YOUTU_GGUF_REPO="${YOUTU_GGUF_REPO:-tencent/Youtu-LLM-2B-GGUF}"
YOUTU_GGUF_FILE="${YOUTU_GGUF_FILE:-Youtu-LLM-2B-Q8_0.gguf}"
BRAIN_MODEL_CACHE_DIR="${BRAIN_MODEL_CACHE_DIR:-${YOUTU_CACHE_DIR:-$BRAIN_DIR/model-cache}}"
BRAIN_VLLM_PORT="${BRAIN_VLLM_PORT:-${YOUTU_VLLM_PORT:-8000}}"
BRAIN_LLAMA_CPP_PORT="${BRAIN_LLAMA_CPP_PORT:-${YOUTU_LLAMA_CPP_PORT:-8081}}"
AUTO_START_MODEL_SERVICE="${AUTO_START_MODEL_SERVICE:-1}"
AUTO_INSTALL_TRIGGER_DEPS="${AUTO_INSTALL_TRIGGER_DEPS:-0}"
CHEAP_COGNITION_RUNTIME="${CHEAP_COGNITION_RUNTIME:-auto}"
DRY_RUN=0
PKG_MANAGER=""
SELECTED_RUNTIME=""
SELECTED_MODEL=""
SELECTED_BASE_URL=""
GENERATED_ENV_FILE="${SPIDERWEB_DIR}/.generated/spiderweb-runtime.env"
SPIDERWEB_HOME_DIR="${SPIDERWEB_HOME_DIR:-$HOME/.spiderweb}"
HF_HOME_DIR="${HF_HOME_DIR:-$SPIDERWEB_HOME_DIR/hf}"
HF_HUB_CACHE_DIR="${HF_HUB_CACHE_DIR:-$HF_HOME_DIR/hub}"

log() { printf "[INFO] %s\n" "$*"; }
warn() { printf "[WARN] %s\n" "$*" >&2; }
err() { printf "[ERROR] %s\n" "$*" >&2; }
have_cmd() { command -v "$1" >/dev/null 2>&1; }

brain_vllm_patches_ready() {
  local patches_dir="$SCRIPT_DIR/../infra/vllm/patches"
  local required=(
    "youtu_llm.py"
    "configuration_youtu.py"
    "registry.py"
    "__init__.py"
  )
  local file=""
  for file in "${required[@]}"; do
    if [ ! -f "$patches_dir/$file" ]; then
      return 1
    fi
  done
  return 0
}

usage() {
  cat <<'EOF'
Spiderweb full-stack bootstrapper

Usage:
  ./install_spiderweb_stack.sh [options]

Options:
  --dry-run   Validate config and print planned actions without changing the system
  --help      Show this help
EOF
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --dry-run)
        DRY_RUN=1
        ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        err "Unknown argument: $1"
        usage
        exit 1
        ;;
    esac
    shift
  done
}

run_cmd() {
  if [ "$DRY_RUN" -eq 1 ]; then
    printf "[DRY-RUN] %s\n" "$*"
    return 0
  fi
  eval "$@"
}

detect_package_manager() {
  if have_cmd apt-get; then PKG_MANAGER="apt-get"; return; fi
  if have_cmd dnf; then PKG_MANAGER="dnf"; return; fi
  if have_cmd yum; then PKG_MANAGER="yum"; return; fi
  if have_cmd zypper; then PKG_MANAGER="zypper"; return; fi
  if have_cmd pacman; then PKG_MANAGER="pacman"; return; fi
  if have_cmd apk; then PKG_MANAGER="apk"; return; fi
  if have_cmd brew; then PKG_MANAGER="brew"; return; fi
  PKG_MANAGER=""
}

install_packages() {
  local packages=(git make curl tar python3)

  if ! have_cmd node; then
    packages+=(nodejs)
  fi
  if ! have_cmd npm; then
    packages+=(npm)
  fi
  if ! have_cmd pip3; then
    packages+=(python3-pip)
  fi
  if ! have_cmd cmake; then
    packages+=(cmake)
  fi
  if ! have_cmd c++; then
    packages+=(g++)
  fi

  case "$PKG_MANAGER" in
    apt-get)
      run_cmd "sudo apt-get update"
      run_cmd "sudo apt-get install -y ${packages[*]}"
      ;;
    dnf)
      run_cmd "sudo dnf install -y ${packages[*]}"
      ;;
    yum)
      run_cmd "sudo yum install -y ${packages[*]}"
      ;;
    zypper)
      run_cmd "sudo zypper install -y ${packages[*]}"
      ;;
    pacman)
      run_cmd "sudo pacman -Sy --noconfirm ${packages[*]}"
      ;;
    apk)
      run_cmd "sudo apk add ${packages[*]}"
      ;;
    brew)
      run_cmd "brew install ${packages[*]}"
      ;;
    *)
      warn "No supported package manager detected. Install required packages manually: ${packages[*]}"
      ;;
  esac
}

install_spiderweb_core() {
  log "Installing Spiderweb core through install_spiderweb.sh"
  if [ "$DRY_RUN" -eq 1 ]; then
    run_cmd "cd '$SCRIPT_DIR' && ./install_spiderweb.sh --dry-run"
  else
    run_cmd "cd '$SCRIPT_DIR' && ./install_spiderweb.sh"
  fi
}

prepare_directories() {
  run_cmd "mkdir -p '$BRAIN_DIR' '$BRAIN_MODEL_CACHE_DIR' '$HF_HOME_DIR' '$HF_HUB_CACHE_DIR' '$(dirname "$GENERATED_ENV_FILE")'"
}

install_python_model_tools() {
  if have_cmd python3; then
    run_cmd "HF_HOME='$HF_HOME_DIR' HF_HUB_CACHE='$HF_HUB_CACHE_DIR' python3 -m pip install --upgrade pip huggingface_hub"
  else
    warn "python3 not available; cannot install huggingface_hub automatically"
  fi
}

has_suitable_gpu() {
  if ! have_cmd nvidia-smi; then
    return 1
  fi
  nvidia-smi >/dev/null 2>&1 || return 1
  return 0
}

select_runtime() {
  case "$CHEAP_COGNITION_RUNTIME" in
    vllm|llama_cpp)
      SELECTED_RUNTIME="$CHEAP_COGNITION_RUNTIME"
      ;;
    auto)
      if has_suitable_gpu; then
        SELECTED_RUNTIME="vllm"
      else
        SELECTED_RUNTIME="llama_cpp"
      fi
      ;;
    *)
      err "Unsupported CHEAP_COGNITION_RUNTIME: $CHEAP_COGNITION_RUNTIME"
      exit 1
      ;;
  esac

  if [ "$SELECTED_RUNTIME" = "vllm" ] && ! brain_vllm_patches_ready; then
    if [ "$CHEAP_COGNITION_RUNTIME" = "vllm" ]; then
      err "Native vLLM requires the Youtu integration files under $SCRIPT_DIR/../infra/vllm/patches/"
      err "Missing one or more of: youtu_llm.py, configuration_youtu.py, registry.py, __init__.py"
      exit 1
    fi
    warn "Native vLLM patches are missing; falling back to llama.cpp"
    SELECTED_RUNTIME="llama_cpp"
  fi

  if [ "$SELECTED_RUNTIME" = "vllm" ]; then
    SELECTED_MODEL="$YOUTU_MODEL_REPO"
    SELECTED_BASE_URL="http://127.0.0.1:${BRAIN_VLLM_PORT}/v1"
  else
    SELECTED_MODEL="$YOUTU_GGUF_REPO:$YOUTU_GGUF_FILE"
    SELECTED_BASE_URL="http://127.0.0.1:${BRAIN_LLAMA_CPP_PORT}/v1"
  fi

  log "Selected cheap-cognition runtime: $SELECTED_RUNTIME"
  log "Selected model source: $SELECTED_MODEL"
}

download_vllm_model() {
  if [ -z "${HF_TOKEN:-}" ]; then
    warn "HF_TOKEN is not set. Skipping automated Hugging Face model pull for vLLM path."
    return 0
  fi
  run_cmd "HF_HOME='$HF_HOME_DIR' HF_HUB_CACHE='$HF_HUB_CACHE_DIR' python3 -m huggingface_hub download '$YOUTU_MODEL_REPO' --local-dir '$BRAIN_DIR' --token '$HF_TOKEN'"
}

download_gguf_model() {
  if [ -z "${HF_TOKEN:-}" ]; then
    warn "HF_TOKEN is not set. Skipping automated Hugging Face GGUF pull for llama.cpp path."
    return 0
  fi
  run_cmd "HF_HOME='$HF_HOME_DIR' HF_HUB_CACHE='$HF_HUB_CACHE_DIR' python3 -m huggingface_hub download '$YOUTU_GGUF_REPO' '$YOUTU_GGUF_FILE' --local-dir '$BRAIN_DIR' --token '$HF_TOKEN'"
}

install_trigger_dependencies() {
  if [ ! -d "$TRIGGER_DIR" ]; then
    warn "Trigger workspace not found at $TRIGGER_DIR"
    return 0
  fi
  if [ "$AUTO_INSTALL_TRIGGER_DEPS" != "1" ]; then
    log "Skipping Trigger.dev dependency install because AUTO_INSTALL_TRIGGER_DEPS=$AUTO_INSTALL_TRIGGER_DEPS"
    return 0
  fi
  run_cmd "cd '$TRIGGER_DIR' && npm install"
}

install_llama_cpp() {
  local llama_dir="$SPIDERWEB_DIR/.cache/llama.cpp"
  local build_dir="$llama_dir/build"

  if have_cmd llama-server; then
    return 0
  fi

  run_cmd "mkdir -p '$SPIDERWEB_DIR/.cache'"
  if [ ! -d "$llama_dir/.git" ]; then
    run_cmd "git clone https://github.com/ggml-org/llama.cpp '$llama_dir'"
  else
    run_cmd "cd '$llama_dir' && git pull --ff-only"
  fi

  run_cmd "cmake -S '$llama_dir' -B '$build_dir' -DLLAMA_BUILD_SERVER=ON"
  run_cmd "cmake --build '$build_dir' -j"
  run_cmd "mkdir -p '$INSTALL_PREFIX/bin'"
  run_cmd "cp '$build_dir/bin/llama-server' '$INSTALL_PREFIX/bin/llama-server'"
}

start_llama_cpp_server() {
  if [ "$AUTO_START_MODEL_SERVICE" != "1" ]; then
    log "Skipping model service startup because AUTO_START_MODEL_SERVICE=$AUTO_START_MODEL_SERVICE"
    return 0
  fi

  local gguf_path="$BRAIN_DIR/$YOUTU_GGUF_FILE"
  local pid_file="$BRAIN_DIR/llama-server.pid"

  if [ ! -f "$gguf_path" ]; then
    warn "GGUF file not found at $gguf_path; llama.cpp server cannot start yet"
    return 0
  fi

  run_cmd "if [ -f '$pid_file' ] && kill -0 \"\$(cat '$pid_file')\" 2>/dev/null; then exit 0; fi"
  run_cmd "nohup '$INSTALL_PREFIX/bin/llama-server' -m '$gguf_path' --host 0.0.0.0 --port '$BRAIN_LLAMA_CPP_PORT' --log-disable > '$BRAIN_DIR/llama-server.log' 2>&1 & echo \$! > '$pid_file'"
}

write_generated_env() {
  run_cmd "cat > '$GENERATED_ENV_FILE' <<EOF
SPIDERWEB_INTAKE_CHEAP_COGNITION_ENABLED=1
SPIDERWEB_INTAKE_CHEAP_COGNITION_RUNTIME=$SELECTED_RUNTIME
SPIDERWEB_INTAKE_CHEAP_COGNITION_BASE_URL=$SELECTED_BASE_URL
SPIDERWEB_INTAKE_CHEAP_COGNITION_MODEL=${YOUTU_MODEL_REPO}
SPIDERWEB_INTAKE_CHEAP_COGNITION_API_KEY=
SPIDERWEB_INTAKE_CHEAP_COGNITION_TIMEOUT_SECONDS=30
HF_HOME=$HF_HOME_DIR
HF_HUB_CACHE=$HF_HUB_CACHE_DIR
EOF"
}

prepare_runtime() {
  if [ "$SELECTED_RUNTIME" = "vllm" ]; then
    download_vllm_model
    warn "Legacy bootstrap_runtime.sh no longer manages Docker containers; use the native vLLM start helpers instead."
  else
    download_gguf_model
    install_llama_cpp
    start_llama_cpp_server
  fi
}

print_next_steps() {
  cat <<EOF

Spiderweb stack bootstrap completed.

Autonomous runtime decision:
- runtime: ${SELECTED_RUNTIME}
- base URL: ${SELECTED_BASE_URL}
- generated env: ${GENERATED_ENV_FILE}
- HF cache home: ${HF_HOME_DIR}
- HF hub cache: ${HF_HUB_CACHE_DIR}

Expected next steps:
- source the generated env file before running Spiderweb services if needed
- verify the local model endpoint responds on ${SELECTED_BASE_URL}
- configure Trigger.dev credentials in ${TRIGGER_DIR}/.env
- run Spiderweb wakeup with:
  sweb wakeup

If Spiderweb is not on PATH yet, try:
  ${INSTALL_PREFIX}/bin/sweb wakeup
EOF
}

main() {
  parse_args "$@"
  detect_package_manager
  install_packages
  install_spiderweb_core
  prepare_directories
  install_python_model_tools
  install_trigger_dependencies
  select_runtime
  prepare_runtime
  write_generated_env
  print_next_steps
}

main "$@"
