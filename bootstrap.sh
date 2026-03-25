#!/usr/bin/env bash
set -euo pipefail

# Spiderweb bootstrap entrypoint
# This is the single user-facing install command.
# Helper scripts under scripts/ are internal chapters only.
# This root entrypoint runs the full setup book from front to back.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_CONFIG_FILE="${SCRIPT_DIR}/bootstrap.conf"
BOOTSTRAP_CONFIG="${BOOTSTRAP_CONFIG:-$DEFAULT_CONFIG_FILE}"

if [ -f "$BOOTSTRAP_CONFIG" ]; then
  # shellcheck source=/dev/null
  . "$BOOTSTRAP_CONFIG"
fi

SPIDERWEB_DIR="${SPIDERWEB_DIR:-$HOME/spiderweb}"
INSTALL_PREFIX="${INSTALL_PREFIX:-$HOME/.local}"
TRIGGER_DIR="${TRIGGER_DIR:-$SPIDERWEB_DIR/trigger}"
BRAIN_DIR="${BRAIN_DIR:-${YOUTU_DIR:-$SPIDERWEB_DIR/brain}}"
YOUTU_MODEL_REPO="${YOUTU_MODEL_REPO:-tencent/Youtu-LLM-2B}"
YOUTU_GGUF_REPO="${YOUTU_GGUF_REPO:-tencent/Youtu-LLM-2B-GGUF}"
YOUTU_GGUF_FILE="${YOUTU_GGUF_FILE:-Youtu-LLM-2B-Q8_0.gguf}"
BRAIN_MODEL_CACHE_DIR="${BRAIN_MODEL_CACHE_DIR:-${YOUTU_CACHE_DIR:-$BRAIN_DIR/model-cache}}"
BRAIN_VLLM_VENV="${BRAIN_VLLM_VENV:-${YOUTU_VLLM_VENV:-$BRAIN_DIR/.venv-vllm}}"
BRAIN_VLLM_PORT="${BRAIN_VLLM_PORT:-${YOUTU_VLLM_PORT:-8000}}"
BRAIN_VLLM_HOST="${BRAIN_VLLM_HOST:-${YOUTU_VLLM_HOST:-127.0.0.1}}"
BRAIN_VLLM_PID_FILE="${BRAIN_VLLM_PID_FILE:-${YOUTU_VLLM_PID_FILE:-$BRAIN_DIR/brain-vllm.pid}}"
BRAIN_VLLM_LOG_FILE="${BRAIN_VLLM_LOG_FILE:-${YOUTU_VLLM_LOG_FILE:-$BRAIN_DIR/brain-vllm.log}}"
BRAIN_VLLM_MAX_MODEL_LEN="${BRAIN_VLLM_MAX_MODEL_LEN:-${YOUTU_VLLM_MAX_MODEL_LEN:-32768}}"
BRAIN_VLLM_GPU_MEMORY_UTILIZATION="${BRAIN_VLLM_GPU_MEMORY_UTILIZATION:-${YOUTU_VLLM_GPU_MEMORY_UTILIZATION:-0.85}}"
BRAIN_LLAMA_CPP_PORT="${BRAIN_LLAMA_CPP_PORT:-${YOUTU_LLAMA_CPP_PORT:-8081}}"
AUTO_START_MODEL_SERVICE="${AUTO_START_MODEL_SERVICE:-1}"
ENABLE_TRIGGER_WORKSPACE="${ENABLE_TRIGGER_WORKSPACE:-0}"
AUTO_INSTALL_TRIGGER_DEPS="${AUTO_INSTALL_TRIGGER_DEPS:-0}"
CHEAP_COGNITION_RUNTIME="${CHEAP_COGNITION_RUNTIME:-auto}"
HF_TOKEN="${HF_TOKEN:-}"
TRIGGER_ACCESS_TOKEN="${TRIGGER_ACCESS_TOKEN:-}"
DRY_RUN=0
SHOW_STATE=0
RESET_STATE=0
PKG_MANAGER=""
SELECTED_RUNTIME=""
SELECTED_MODEL=""
SELECTED_BASE_URL=""
SPIDERWEB_HOME_DIR="${SPIDERWEB_HOME_DIR:-$HOME/.spiderweb}"
HF_HOME_DIR="${HF_HOME_DIR:-$SPIDERWEB_HOME_DIR/hf}"
HF_HUB_CACHE_DIR="${HF_HUB_CACHE_DIR:-$HF_HOME_DIR/hub}"
GENERATED_ENV_FILE="${SPIDERWEB_DIR}/.generated/spiderweb-runtime.env"
RUNTIME_ENV_FILE="${RUNTIME_ENV_FILE:-$SPIDERWEB_HOME_DIR/runtime.env}"
SETUP_NOTES_FILE="${SETUP_NOTES_FILE:-$SPIDERWEB_HOME_DIR/setup-notes.txt}"
BOOTSTRAP_STATE_FILE="${BOOTSTRAP_STATE_FILE:-$SPIDERWEB_HOME_DIR/bootstrap-state.env}"
BOOTSTRAP_LAST_STATUS="${BOOTSTRAP_LAST_STATUS:-new}"

log() { printf "[INFO] %s\n" "$*"; }
warn() { printf "[WARN] %s\n" "$*" >&2; }
err() { printf "[ERROR] %s\n" "$*" >&2; }
have_cmd() { command -v "$1" >/dev/null 2>&1; }
is_interactive() { [ -t 0 ] && [ -t 1 ]; }

brain_vllm_patches_ready() {
  local patches_dir="$SCRIPT_DIR/infra/vllm/patches"
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

if [ -f "$BOOTSTRAP_STATE_FILE" ]; then
  # shellcheck source=/dev/null
  . "$BOOTSTRAP_STATE_FILE"
fi

BRAIN_DIR="${BRAIN_DIR:-${YOUTU_DIR:-$SPIDERWEB_DIR/brain}}"

usage() {
  cat <<'EOF'
Spiderweb bootstrap

This is the single user-facing setup command.
Internal helpers under scripts/ are invoked automatically as chapters.

Usage:
  ./bootstrap.sh [options]

Options:
  --dry-run   Validate config and print planned actions without changing the system
  --show-state  Print saved bootstrap state and exit
  --reset-state Clear saved bootstrap state and restart from zero
  --help      Show this help
EOF
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --dry-run)
        DRY_RUN=1
        ;;
      --show-state)
        SHOW_STATE=1
        ;;
      --reset-state)
        RESET_STATE=1
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

clear_bootstrap_state() {
  rm -f "$BOOTSTRAP_STATE_FILE"
  CHAPTER_PREINSTALL_DONE=0
  CHAPTER_PACKAGES_DONE=0
  CHAPTER_CORE_DONE=0
  CHAPTER_DIRECTORIES_DONE=0
  CHAPTER_TRIGGER_DONE=0
  CHAPTER_RUNTIME_SELECT_DONE=0
  CHAPTER_MODEL_RUNTIME_DONE=0
  CHAPTER_FINALIZE_DONE=0
  BOOTSTRAP_LAST_STATUS="new"
}

print_bootstrap_state() {
  if [ ! -f "$BOOTSTRAP_STATE_FILE" ]; then
    printf "No saved bootstrap state at %s\n" "$BOOTSTRAP_STATE_FILE"
    return 0
  fi

  cat <<EOF
Spiderweb bootstrap state
- state file: $BOOTSTRAP_STATE_FILE
- status: ${BOOTSTRAP_LAST_STATUS}
- selected runtime: ${SELECTED_RUNTIME:-unset}
- selected model: ${SELECTED_MODEL:-unset}
- selected base URL: ${SELECTED_BASE_URL:-unset}
- chapters:
  - preinstall: ${CHAPTER_PREINSTALL_DONE:-0}
  - packages: ${CHAPTER_PACKAGES_DONE:-0}
  - core: ${CHAPTER_CORE_DONE:-0}
  - directories: ${CHAPTER_DIRECTORIES_DONE:-0}
  - trigger: ${CHAPTER_TRIGGER_DONE:-0}
  - runtime_select: ${CHAPTER_RUNTIME_SELECT_DONE:-0}
  - model_runtime: ${CHAPTER_MODEL_RUNTIME_DONE:-0}
  - finalize: ${CHAPTER_FINALIZE_DONE:-0}
EOF
}

print_existing_bootstrap_summary() {
  cat <<EOF

Spiderweb bootstrap is already complete from the saved state.

- runtime: ${SELECTED_RUNTIME:-unknown}
- base URL: ${SELECTED_BASE_URL:-unknown}
- runtime env: ${RUNTIME_ENV_FILE}
- setup notes: ${SETUP_NOTES_FILE}
- bootstrap state: ${BOOTSTRAP_STATE_FILE}

If you want to rebuild from zero, rerun with:
  ./bootstrap.sh --reset-state
EOF
}

save_bootstrap_state() {
  if [ "$DRY_RUN" -eq 1 ]; then
    return 0
  fi

  mkdir -p "$(dirname "$BOOTSTRAP_STATE_FILE")"
  umask 077
  {
    printf 'SPIDERWEB_DIR=%q\n' "$SPIDERWEB_DIR"
    printf 'SPIDERWEB_HOME_DIR=%q\n' "$SPIDERWEB_HOME_DIR"
    printf 'HF_HOME_DIR=%q\n' "$HF_HOME_DIR"
    printf 'HF_HUB_CACHE_DIR=%q\n' "$HF_HUB_CACHE_DIR"
    printf 'INSTALL_PREFIX=%q\n' "$INSTALL_PREFIX"
    printf 'TRIGGER_DIR=%q\n' "$TRIGGER_DIR"
    printf 'BRAIN_DIR=%q\n' "$BRAIN_DIR"
    printf 'YOUTU_MODEL_REPO=%q\n' "$YOUTU_MODEL_REPO"
    printf 'YOUTU_GGUF_REPO=%q\n' "$YOUTU_GGUF_REPO"
    printf 'YOUTU_GGUF_FILE=%q\n' "$YOUTU_GGUF_FILE"
    printf 'BRAIN_MODEL_CACHE_DIR=%q\n' "$BRAIN_MODEL_CACHE_DIR"
    printf 'BRAIN_VLLM_VENV=%q\n' "$BRAIN_VLLM_VENV"
    printf 'BRAIN_VLLM_PORT=%q\n' "$BRAIN_VLLM_PORT"
    printf 'BRAIN_VLLM_HOST=%q\n' "$BRAIN_VLLM_HOST"
    printf 'BRAIN_VLLM_PID_FILE=%q\n' "$BRAIN_VLLM_PID_FILE"
    printf 'BRAIN_VLLM_LOG_FILE=%q\n' "$BRAIN_VLLM_LOG_FILE"
    printf 'BRAIN_VLLM_MAX_MODEL_LEN=%q\n' "$BRAIN_VLLM_MAX_MODEL_LEN"
    printf 'BRAIN_VLLM_GPU_MEMORY_UTILIZATION=%q\n' "$BRAIN_VLLM_GPU_MEMORY_UTILIZATION"
    printf 'BRAIN_LLAMA_CPP_PORT=%q\n' "$BRAIN_LLAMA_CPP_PORT"
    printf 'AUTO_START_MODEL_SERVICE=%q\n' "$AUTO_START_MODEL_SERVICE"
    printf 'ENABLE_TRIGGER_WORKSPACE=%q\n' "$ENABLE_TRIGGER_WORKSPACE"
    printf 'AUTO_INSTALL_TRIGGER_DEPS=%q\n' "$AUTO_INSTALL_TRIGGER_DEPS"
    printf 'CHEAP_COGNITION_RUNTIME=%q\n' "$CHEAP_COGNITION_RUNTIME"
    printf 'HF_TOKEN=%q\n' "$HF_TOKEN"
    printf 'TRIGGER_ACCESS_TOKEN=%q\n' "$TRIGGER_ACCESS_TOKEN"
    printf 'PKG_MANAGER=%q\n' "$PKG_MANAGER"
    printf 'SELECTED_RUNTIME=%q\n' "$SELECTED_RUNTIME"
    printf 'SELECTED_MODEL=%q\n' "$SELECTED_MODEL"
    printf 'SELECTED_BASE_URL=%q\n' "$SELECTED_BASE_URL"
    printf 'GENERATED_ENV_FILE=%q\n' "$GENERATED_ENV_FILE"
    printf 'RUNTIME_ENV_FILE=%q\n' "$RUNTIME_ENV_FILE"
    printf 'SETUP_NOTES_FILE=%q\n' "$SETUP_NOTES_FILE"
    printf 'BOOTSTRAP_STATE_FILE=%q\n' "$BOOTSTRAP_STATE_FILE"
    printf 'BOOTSTRAP_LAST_STATUS=%q\n' "$BOOTSTRAP_LAST_STATUS"
    printf 'CHAPTER_PREINSTALL_DONE=%q\n' "${CHAPTER_PREINSTALL_DONE:-0}"
    printf 'CHAPTER_PACKAGES_DONE=%q\n' "${CHAPTER_PACKAGES_DONE:-0}"
    printf 'CHAPTER_CORE_DONE=%q\n' "${CHAPTER_CORE_DONE:-0}"
    printf 'CHAPTER_DIRECTORIES_DONE=%q\n' "${CHAPTER_DIRECTORIES_DONE:-0}"
    printf 'CHAPTER_TRIGGER_DONE=%q\n' "${CHAPTER_TRIGGER_DONE:-0}"
    printf 'CHAPTER_RUNTIME_SELECT_DONE=%q\n' "${CHAPTER_RUNTIME_SELECT_DONE:-0}"
    printf 'CHAPTER_MODEL_RUNTIME_DONE=%q\n' "${CHAPTER_MODEL_RUNTIME_DONE:-0}"
    printf 'CHAPTER_FINALIZE_DONE=%q\n' "${CHAPTER_FINALIZE_DONE:-0}"
  } > "$BOOTSTRAP_STATE_FILE"
}

mark_chapter_done() {
  local chapter_var="$1"
  printf -v "$chapter_var" '%s' "1"
  BOOTSTRAP_LAST_STATUS="in_progress"
  save_bootstrap_state
}

chapter_is_done() {
  local chapter_var="$1"
  [ "${!chapter_var:-0}" = "1" ]
}

run_chapter() {
  local chapter_var="$1"
  local chapter_label="$2"
  local chapter_fn="$3"

  if chapter_is_done "$chapter_var"; then
    log "Skipping completed chapter: $chapter_label"
    return 0
  fi

  "$chapter_fn"
  mark_chapter_done "$chapter_var"
}

append_setup_note() {
  local line="$1"
  run_cmd "mkdir -p '$(dirname "$SETUP_NOTES_FILE")'"
  run_cmd "printf '%s\n' \"$line\" >> '$SETUP_NOTES_FILE'"
}

prompt_secret_if_missing() {
  local var_name="$1"
  local prompt_label="$2"
  local guidance="$3"
  local current_value="${!var_name:-}"

  if [ -n "$current_value" ]; then
    return 0
  fi
  if [ "$DRY_RUN" -eq 1 ]; then
    warn "$prompt_label not set."
    warn "$guidance"
    return 0
  fi
  if ! is_interactive; then
    warn "$prompt_label not set."
    warn "$guidance"
    return 0
  fi

  printf "\n%s\n" "$prompt_label"
  printf "%s\n" "$guidance"
  printf "Enter value now, or press Enter to skip: "
  # shellcheck disable=SC2162
  read -r input_value
  if [ -n "$input_value" ]; then
    printf -v "$var_name" '%s' "$input_value"
    export "$var_name"
  fi
}

chapter_preinstall_feed() {
  log "Chapter 1/8: pre-install feed"
  run_cmd ": > '$SETUP_NOTES_FILE'"

  prompt_secret_if_missing \
    "HF_TOKEN" \
    "Optional Hugging Face token" \
    "Public Youtu repos usually download without a token. If you use gated Hugging Face assets later, create a token at https://huggingface.co/settings/tokens and paste it here."

  if [ "$ENABLE_TRIGGER_WORKSPACE" = "1" ]; then
    prompt_secret_if_missing \
      "TRIGGER_ACCESS_TOKEN" \
      "Optional Trigger.dev access token" \
      "Only needed if your Trigger workflow requires authenticated Trigger.dev access. Create it from your Trigger.dev account settings if you plan to enable that workspace."
  fi

  append_setup_note "Spiderweb bootstrap setup notes"
  append_setup_note ""
  append_setup_note "Provider API keys are not part of bootstrap itself."
  append_setup_note "You will still need to configure at least one main LLM provider before full agent use."
  append_setup_note "Recommended next step after install:"
  append_setup_note "- edit ~/.spiderweb/config.json or the active Spiderweb config"
  append_setup_note "- add a provider entry under model_list with api_key and api_base as needed"
  append_setup_note ""
  append_setup_note "Helpful key sources:"
  append_setup_note "- OpenAI: https://platform.openai.com/api-keys"
  append_setup_note "- OpenRouter: https://openrouter.ai/keys"
  append_setup_note "- Zhipu: https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys"
  append_setup_note "- Hugging Face: https://huggingface.co/settings/tokens"

  if [ -z "$HF_TOKEN" ]; then
    append_setup_note ""
    append_setup_note "HF_TOKEN was not supplied during bootstrap."
    append_setup_note "This is acceptable for the public Youtu repos unless access rules change."
  fi
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

chapter_install_packages() {
  log "Chapter 2/8: install system packages"
  local packages=(git make curl tar python3)

  if ! have_cmd git-lfs; then packages+=(git-lfs); fi
  if ! have_cmd node; then packages+=(nodejs); fi
  if ! have_cmd npm; then packages+=(npm); fi
  if ! have_cmd pip3; then packages+=(python3-pip); fi
  if ! have_cmd python3 || ! python3 -m venv --help >/dev/null 2>&1; then packages+=(python3-venv); fi
  if ! have_cmd cmake; then packages+=(cmake); fi
  if ! have_cmd c++; then packages+=(g++); fi

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
      warn "No supported package manager detected. Install manually if required: ${packages[*]}"
      ;;
  esac

  if have_cmd git && have_cmd git-lfs; then
    run_cmd "git lfs install --skip-repo"
  fi
}

chapter_install_spiderweb_core() {
  log "Chapter 3/8: install Spiderweb core"
  if [ "$DRY_RUN" -eq 1 ]; then
    run_cmd "cd '$SCRIPT_DIR/scripts' && ./install_core.sh --dry-run"
  else
    run_cmd "cd '$SCRIPT_DIR/scripts' && ./install_core.sh"
  fi
}

chapter_prepare_directories() {
  log "Chapter 4/8: prepare local directories"
  run_cmd "mkdir -p '$BRAIN_DIR' '$BRAIN_MODEL_CACHE_DIR' '$SPIDERWEB_HOME_DIR' '$HF_HOME_DIR' '$HF_HUB_CACHE_DIR' '$(dirname "$GENERATED_ENV_FILE")' '$(dirname "$RUNTIME_ENV_FILE")' '$(dirname "$SETUP_NOTES_FILE")'"
}

chapter_prepare_trigger() {
  log "Chapter 5/8: prepare optional Trigger workspace"
  if have_cmd python3; then
    run_cmd "HF_HOME='$HF_HOME_DIR' HF_HUB_CACHE='$HF_HUB_CACHE_DIR' python3 -m pip install --upgrade pip huggingface_hub"
  fi
  if [ "$ENABLE_TRIGGER_WORKSPACE" != "1" ]; then
    log "Skipping Trigger workspace setup because ENABLE_TRIGGER_WORKSPACE=$ENABLE_TRIGGER_WORKSPACE"
    return 0
  fi
  if [ -d "$TRIGGER_DIR" ] && [ "$AUTO_INSTALL_TRIGGER_DEPS" = "1" ]; then
    run_cmd "cd '$TRIGGER_DIR' && npm install"
  fi
}

has_suitable_gpu() {
  if ! have_cmd nvidia-smi; then return 1; fi
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
      err "Native vLLM requires the Youtu integration files under $SCRIPT_DIR/infra/vllm/patches/"
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
}

chapter_select_runtime() {
  log "Chapter 6/8: inspect host and select runtime"
  select_runtime
  log "Selected runtime: $SELECTED_RUNTIME"
  log "Selected model source: $SELECTED_MODEL"
}

download_vllm_model() {
  local cmd="HF_HOME='$HF_HOME_DIR' HF_HUB_CACHE='$HF_HUB_CACHE_DIR' python3 -m huggingface_hub download '$YOUTU_MODEL_REPO' --local-dir '$BRAIN_DIR'"
  if [ -n "$HF_TOKEN" ]; then
    cmd="$cmd --token '$HF_TOKEN'"
  fi
  run_cmd "$cmd"
}

download_gguf_model() {
  local cmd="HF_HOME='$HF_HOME_DIR' HF_HUB_CACHE='$HF_HUB_CACHE_DIR' python3 -m huggingface_hub download '$YOUTU_GGUF_REPO' '$YOUTU_GGUF_FILE' --local-dir '$BRAIN_DIR'"
  if [ -n "$HF_TOKEN" ]; then
    cmd="$cmd --token '$HF_TOKEN'"
  fi
  run_cmd "$cmd"
}

install_vllm_runtime() {
  run_cmd "python3 -m venv '$BRAIN_VLLM_VENV'"
  run_cmd "'$BRAIN_VLLM_VENV/bin/python' -m pip install --upgrade pip setuptools wheel"
  run_cmd "HF_HOME='$HF_HOME_DIR' HF_HUB_CACHE='$HF_HUB_CACHE_DIR' '$BRAIN_VLLM_VENV/bin/python' -m pip install 'vllm==0.10.2' 'huggingface_hub'"
}

start_vllm_server() {
  if [ "$AUTO_START_MODEL_SERVICE" != "1" ]; then
    log "Skipping model service startup because AUTO_START_MODEL_SERVICE=$AUTO_START_MODEL_SERVICE"
    return 0
  fi
  run_cmd "cd '$SCRIPT_DIR' && HF_HOME='$HF_HOME_DIR' HF_HUB_CACHE='$HF_HUB_CACHE_DIR' BRAIN_DIR='$BRAIN_DIR' BRAIN_MODEL_CACHE_DIR='$BRAIN_MODEL_CACHE_DIR' BRAIN_VLLM_VENV='$BRAIN_VLLM_VENV' BRAIN_VLLM_PORT='$BRAIN_VLLM_PORT' BRAIN_VLLM_HOST='$BRAIN_VLLM_HOST' BRAIN_VLLM_PID_FILE='$BRAIN_VLLM_PID_FILE' BRAIN_VLLM_LOG_FILE='$BRAIN_VLLM_LOG_FILE' BRAIN_VLLM_MODEL_PATH='$BRAIN_DIR' BRAIN_VLLM_MAX_MODEL_LEN='$BRAIN_VLLM_MAX_MODEL_LEN' BRAIN_VLLM_GPU_MEMORY_UTILIZATION='$BRAIN_VLLM_GPU_MEMORY_UTILIZATION' ./scripts/start_brain_vllm.sh"
}

install_llama_cpp() {
  local llama_dir="$SPIDERWEB_DIR/.cache/llama.cpp"
  local build_dir="$llama_dir/build"

  if have_cmd llama-server; then return 0; fi

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

chapter_prepare_model_runtime() {
  log "Chapter 7/8: download model and prepare local runtime"
  if [ "$SELECTED_RUNTIME" = "vllm" ]; then
    download_vllm_model
    install_vllm_runtime
    start_vllm_server
  else
    download_gguf_model
    install_llama_cpp
    start_llama_cpp_server
  fi
}

write_runtime_env_file() {
  local target_file="$1"
  local model_value="$SELECTED_MODEL"

  run_cmd "cat > '$target_file' <<EOF
SPIDERWEB_INTAKE_CHEAP_COGNITION_ENABLED=1
SPIDERWEB_INTAKE_CHEAP_COGNITION_RUNTIME=$SELECTED_RUNTIME
SPIDERWEB_INTAKE_CHEAP_COGNITION_BASE_URL=$SELECTED_BASE_URL
SPIDERWEB_INTAKE_CHEAP_COGNITION_MODEL=$model_value
SPIDERWEB_INTAKE_CHEAP_COGNITION_API_KEY=
SPIDERWEB_INTAKE_CHEAP_COGNITION_TIMEOUT_SECONDS=30
HF_HOME=$HF_HOME_DIR
HF_HUB_CACHE=$HF_HUB_CACHE_DIR
EOF"
}

write_generated_env() {
  write_runtime_env_file "$GENERATED_ENV_FILE"
  write_runtime_env_file "$RUNTIME_ENV_FILE"
}

chapter_finalize() {
  log "Chapter 8/8: finalize runtime configuration"
  write_generated_env
  BOOTSTRAP_LAST_STATUS="completed"
  save_bootstrap_state
  cat <<EOF

Spiderweb bootstrap completed.

Autonomous runtime decision:
- runtime: ${SELECTED_RUNTIME}
- base URL: ${SELECTED_BASE_URL}
- generated env: ${GENERATED_ENV_FILE}
- runtime env: ${RUNTIME_ENV_FILE}
- HF cache home: ${HF_HOME_DIR}
- HF hub cache: ${HF_HUB_CACHE_DIR}
- setup notes: ${SETUP_NOTES_FILE}
- bootstrap state: ${BOOTSTRAP_STATE_FILE}

System should now be ready to move toward launch with:
  sweb wakeup

If Spiderweb is not on PATH yet, try:
  ${INSTALL_PREFIX}/bin/sweb wakeup
EOF
}

main() {
  parse_args "$@"
  if [ "$RESET_STATE" -eq 1 ]; then
    log "Resetting saved bootstrap state"
    clear_bootstrap_state
  fi
  if [ "$SHOW_STATE" -eq 1 ]; then
    print_bootstrap_state
    exit 0
  fi
  detect_package_manager
  if [ -f "$BOOTSTRAP_STATE_FILE" ]; then
    log "Found previous bootstrap state at $BOOTSTRAP_STATE_FILE"
    log "Previous bootstrap status: ${BOOTSTRAP_LAST_STATUS}"
  fi
  if [ "${CHAPTER_FINALIZE_DONE:-0}" = "1" ] && [ "$RESET_STATE" -ne 1 ]; then
    print_existing_bootstrap_summary
    exit 0
  fi
  save_bootstrap_state
  run_chapter "CHAPTER_PREINSTALL_DONE" "pre-install feed" chapter_preinstall_feed
  run_chapter "CHAPTER_PACKAGES_DONE" "install system packages" chapter_install_packages
  run_chapter "CHAPTER_CORE_DONE" "install Spiderweb core" chapter_install_spiderweb_core
  run_chapter "CHAPTER_DIRECTORIES_DONE" "prepare local directories" chapter_prepare_directories
  run_chapter "CHAPTER_TRIGGER_DONE" "prepare optional Trigger workspace" chapter_prepare_trigger
  run_chapter "CHAPTER_RUNTIME_SELECT_DONE" "inspect host and select runtime" chapter_select_runtime
  run_chapter "CHAPTER_MODEL_RUNTIME_DONE" "download model and prepare local runtime" chapter_prepare_model_runtime
  run_chapter "CHAPTER_FINALIZE_DONE" "finalize runtime configuration" chapter_finalize
}

main "$@"
