#!/usr/bin/env bash
set -euo pipefail

# Spiderweb installer for Linux hosts (including DirectAdmin/LiteSpeed setups)
# - Installs required tools: git, make, curl/wget, tar, and Go
# - Clones spiderweb
# - Runs make deps, make build, and make install

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_CONFIG_FILE="${SCRIPT_DIR}/../install_spiderweb.conf"

# Override config file location with: INSTALL_CONFIG=/path/to/config ./scripts/install_core.sh
INSTALL_CONFIG="${INSTALL_CONFIG:-$DEFAULT_CONFIG_FILE}"

if [ -f "$INSTALL_CONFIG" ]; then
  # shellcheck source=/dev/null
  . "$INSTALL_CONFIG"
elif [ "$INSTALL_CONFIG" != "$DEFAULT_CONFIG_FILE" ]; then
  printf "[WARN] Installer config file not found: %s\n" "$INSTALL_CONFIG" >&2
fi

SPIDERWEB_REPO="${SPIDERWEB_REPO:-https://github.com/JustSebNL/Spiderweb.git}"
SPIDERWEB_DIR="${SPIDERWEB_DIR:-$HOME/spiderweb}"
INSTALL_PREFIX="${INSTALL_PREFIX:-$HOME/.local}"
MIN_GO_VERSION="${MIN_GO_VERSION:-1.25.0}"
GO_VERSION="${GO_VERSION:-1.25.0}"
GO_INSTALL_DIR="${GO_INSTALL_DIR:-$HOME/.local/go}"
VERIFY_GO_CHECKSUM="${VERIFY_GO_CHECKSUM:-1}"
GO_CHECKSUM="${GO_CHECKSUM:-}"
DRY_RUN=0
PKG_MANAGER=""
LINUX_DISTRO=""
TAR_BIN=""

log() {
  printf "[INFO] %s\n" "$*"
}

get_required_packages() {
  local packages=()

  case "$PKG_MANAGER" in
    apt-get)
      packages=(git make tar ca-certificates)
      ;;
    dnf|yum)
      packages=(git make tar ca-certificates)
      ;;
    zypper)
      packages=(git make tar ca-certificates)
      ;;
    pacman)
      packages=(git make tar ca-certificates)
      ;;
    apk)
      packages=(git make tar ca-certificates)
      ;;
    brew)
      packages=(git make gnu-tar curl)
      ;;
    *)
      packages=(git make tar)
      ;;
  esac

  if ! have_cmd curl && ! have_cmd wget; then
    packages+=(curl)
  fi

  printf '%s\n' "${packages[@]}"
}

warn() {
  printf "[WARN] %s\n" "$*" >&2
}

err() {
  printf "[ERROR] %s\n" "$*" >&2
}

usage() {
  cat <<'EOF'
Spiderweb installer

Usage:
  ./install_spiderweb.sh [options]

Options:
  --dry-run   Validate config and environment without installing anything
  --help      Show this help message

Note:
  This installer must run in a Linux shell.
  On Windows, run it inside WSL.
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

detect_linux_distro() {
  if [ -r /etc/os-release ]; then
    # shellcheck source=/dev/null
    . /etc/os-release
    LINUX_DISTRO="${ID:-unknown}"
  else
    LINUX_DISTRO="unknown"
  fi
}

detect_package_manager() {
  if have_cmd apt-get; then
    PKG_MANAGER="apt-get"
    return
  fi
  if have_cmd dnf; then
    PKG_MANAGER="dnf"
    return
  fi
  if have_cmd yum; then
    PKG_MANAGER="yum"
    return
  fi
  if have_cmd zypper; then
    PKG_MANAGER="zypper"
    return
  fi
  if have_cmd pacman; then
    PKG_MANAGER="pacman"
    return
  fi
  if have_cmd apk; then
    PKG_MANAGER="apk"
    return
  fi
  if have_cmd brew; then
    PKG_MANAGER="brew"
    return
  fi

  PKG_MANAGER=""
}

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

resolve_tar_bin() {
  if have_cmd tar; then
    TAR_BIN="tar"
    return 0
  fi
  if have_cmd gtar; then
    TAR_BIN="gtar"
    return 0
  fi

  TAR_BIN=""
  return 1
}

download_file() {
  local output_file="$1"
  local source_url="$2"

  if have_cmd curl; then
    curl -fsSL -o "$output_file" "$source_url"
    return
  fi

  if have_cmd wget; then
    wget -q -O "$output_file" "$source_url"
    return
  fi

  err "Neither curl nor wget is available to download: $source_url"
  exit 1
}

version_ge() {
  # Returns success if $1 >= $2
  [ "$(printf '%s\n' "$1" "$2" | sort -V | head -n1)" = "$2" ]
}

is_windows_shell() {
  local uname_s
  uname_s="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$uname_s" in
    *mingw*|*msys*|*cygwin*|*windows_nt*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

to_wsl_path() {
  local input_path="$1"
  local normalized drive rest

  normalized="${input_path//\\//}"

  if [[ "$normalized" =~ ^([A-Za-z]):/(.*)$ ]]; then
    drive="${BASH_REMATCH[1],,}"
    rest="${BASH_REMATCH[2]}"
    printf "/mnt/%s/%s" "$drive" "$rest"
    return
  fi

  if [[ "$normalized" =~ ^/([A-Za-z])/(.*)$ ]]; then
    drive="${BASH_REMATCH[1],,}"
    rest="${BASH_REMATCH[2]}"
    printf "/mnt/%s/%s" "$drive" "$rest"
    return
  fi

  printf "%s" "$normalized"
}

print_wsl_routine() {
  local wsl_cmd script_dir_wsl config_wsl

  if have_cmd wsl; then
    wsl_cmd="wsl"
  elif have_cmd wsl.exe; then
    wsl_cmd="wsl.exe"
  else
    err "WSL command not found. Install/enable WSL first, then rerun this installer inside WSL."
    return
  fi

  script_dir_wsl="$(to_wsl_path "$SCRIPT_DIR")"
  config_wsl="$(to_wsl_path "$INSTALL_CONFIG")"

  err "Windows shell detected. Run the installer via WSL using one of these commands:"
  err ""
  err "Dry run:"
  err "  ${wsl_cmd} bash -lc \"cd '${script_dir_wsl}' && INSTALL_CONFIG='${config_wsl}' ./install_spiderweb.sh --dry-run\""
  err ""
  err "Install:"
  err "  ${wsl_cmd} bash -lc \"cd '${script_dir_wsl}' && INSTALL_CONFIG='${config_wsl}' ./install_spiderweb.sh\""
}

list_wsl_distros() {
  local wsl_cmd
  if have_cmd wsl; then
    wsl_cmd="wsl"
  elif have_cmd wsl.exe; then
    wsl_cmd="wsl.exe"
  else
    return 1
  fi

  "$wsl_cmd" -l -q 2>/dev/null | tr -d '\r' | sed '/^[[:space:]]*$/d'
}

get_default_wsl_distro() {
  local wsl_cmd
  if have_cmd wsl; then
    wsl_cmd="wsl"
  elif have_cmd wsl.exe; then
    wsl_cmd="wsl.exe"
  else
    return 1
  fi

  "$wsl_cmd" -l -v 2>/dev/null | tr -d '\r' | awk 'NR>1 && $1=="*" {print $2; exit}'
}

select_wsl_distro() {
  local default_distro selected choice i
  local distros=()

  while IFS= read -r line; do
    distros+=("$line")
  done < <(list_wsl_distros)

  if [ "${#distros[@]}" -eq 0 ]; then
    return 1
  fi

  default_distro="$(get_default_wsl_distro || true)"
  if [ -z "$default_distro" ]; then
    default_distro="${distros[0]}"
  fi

  if [ "${#distros[@]}" -eq 1 ]; then
    printf "%s" "${distros[0]}"
    return 0
  fi

  if [ ! -t 0 ]; then
    printf "%s" "$default_distro"
    return 0
  fi

  printf "Multiple WSL distros detected. Where to install Spiderweb?\n" >&2
  i=1
  for selected in "${distros[@]}"; do
    if [ "$selected" = "$default_distro" ]; then
      printf "  %s. %s (default)\n" "$i" "$selected" >&2
    else
      printf "  %s. %s\n" "$i" "$selected" >&2
    fi
    i=$((i + 1))
  done

  printf "Select distro [1-%s, Enter for default]: " "${#distros[@]}" >&2
  read -r choice

  if [ -z "$choice" ]; then
    printf "%s" "$default_distro"
    return 0
  fi

  if ! printf '%s' "$choice" | grep -Eq '^[0-9]+$'; then
    printf "%s" "$default_distro"
    return 0
  fi

  if [ "$choice" -lt 1 ] || [ "$choice" -gt "${#distros[@]}" ]; then
    printf "%s" "$default_distro"
    return 0
  fi

  printf "%s" "${distros[$((choice - 1))]}"
}

run_in_wsl() {
  local wsl_cmd script_dir_wsl config_wsl selected_distro run_mode

  if have_cmd wsl; then
    wsl_cmd="wsl"
  elif have_cmd wsl.exe; then
    wsl_cmd="wsl.exe"
  else
    print_wsl_routine
    return 1
  fi

  selected_distro="$(select_wsl_distro)"
  if [ -z "$selected_distro" ]; then
    err "No WSL distro found. Install a distro first (for example Ubuntu) and retry."
    return 1
  fi

  script_dir_wsl="$(to_wsl_path "$SCRIPT_DIR")"
  config_wsl="$(to_wsl_path "$INSTALL_CONFIG")"
  run_mode=""
  if [ "$DRY_RUN" = "1" ]; then
    run_mode="--dry-run"
  fi

  err "Windows shell detected. Handing off to WSL distro: $selected_distro"
  "$wsl_cmd" -d "$selected_distro" bash -lc "cd '$script_dir_wsl' && INSTALL_CONFIG='$config_wsl' ./install_spiderweb.sh $run_mode"
}

ensure_linux_runtime() {
  local uname_s handoff_rc
  uname_s="$(uname -s | tr '[:upper:]' '[:lower:]')"
  if [ "$uname_s" != "linux" ]; then
    if is_windows_shell; then
      run_in_wsl
      handoff_rc=$?
      exit "$handoff_rc"
    else
      err "This installer must be run in Linux."
      err "If you are on Windows, run it in WSL (Ubuntu/Debian/etc)."
      exit 1
    fi
  fi
}

get_sudo_prefix() {
  if [ "$(id -u)" -eq 0; then
    printf ""
  elif have_cmd sudo; then
    printf "sudo"
  else
    printf ""
  fi
}

install_with_package_manager() {
  local sudo_prefix
  sudo_prefix="$(get_sudo_prefix)"

  if [ "$PKG_MANAGER" = "apt-get" ]; then
    $sudo_prefix apt-get update
    $sudo_prefix apt-get install -y "$@"
    return
  fi

  if [ "$PKG_MANAGER" = "dnf" ]; then
    $sudo_prefix dnf install -y "$@"
    return
  fi

  if [ "$PKG_MANAGER" = "yum" ]; then
    $sudo_prefix yum install -y "$@"
    return
  fi

  if [ "$PKG_MANAGER" = "zypper" ]; then
    $sudo_prefix zypper --non-interactive install "$@"
    return
  fi

  if [ "$PKG_MANAGER" = "pacman" ]; then
    $sudo_prefix pacman -Sy --noconfirm "$@"
    return
  fi

  if [ "$PKG_MANAGER" = "apk" ]; then
    $sudo_prefix apk add --no-cache "$@"
    return
  fi

  if [ "$PKG_MANAGER" = "brew" ]; then
    brew install "$@"
    return
  fi

  err "No supported package manager found. Please install manually: $*"
  exit 1
}

ensure_basics() {
  local required=()

  detect_linux_distro
  detect_package_manager

  if [ -n "$PKG_MANAGER" ]; then
    log "Detected Linux distro: $LINUX_DISTRO (package manager: $PKG_MANAGER)"
  else
    warn "Detected Linux distro: $LINUX_DISTRO (package manager: unknown)"
  fi

  while IFS= read -r pkg; do
    [ -n "$pkg" ] && required+=("$pkg")
  done < <(get_required_packages)

  if [ "${#required[@]}" -eq 0 ]; then
    return
  fi

  if have_cmd git && have_cmd make && resolve_tar_bin && (have_cmd curl || have_cmd wget); then
    log "System prerequisites already installed (git/make/tar/curl or wget)."
    return
  fi

  if [ -z "$PKG_MANAGER" ]; then
    err "No supported package manager found. Please install manually: ${required[*]}"
    exit 1
  fi

  log "Installing prerequisites for $LINUX_DISTRO via $PKG_MANAGER: ${required[*]}"
  install_with_package_manager "${required[@]}"

  if ! resolve_tar_bin; then
    err "A tar-compatible tool is required but not found (tar/gtar)."
    exit 1
  fi
}

get_go_version() {
  if ! have_cmd go; then
    return 1
  fi

  go version | awk '{print $3}' | sed 's/^go//'
}

ensure_path_exports() {
  local profile
  local line1='export PATH="$HOME/.local/go/bin:$HOME/.local/bin:$PATH"'

  for profile in "$HOME/.bashrc" "$HOME/.profile"; do
    if [ -f "$profile" ] && grep -Fq "$line1" "$profile"; then
      continue
    fi
    printf '\n%s\n' "$line1" >> "$profile"
  done
}

calculate_file_sha256() {
  local target_file="$1"

  if have_cmd sha256sum; then
    sha256sum "$target_file" | awk '{print $1}'
    return
  fi

  if have_cmd shasum; then
    shasum -a 256 "$target_file" | awk '{print $1}'
    return
  fi

  err "No SHA256 tool found (sha256sum or shasum)."
  exit 1
}

verify_go_tarball_checksum() {
  local go_tarball="$1"
  local go_url="$2"
  local tarball_path="$3"
  local expected_checksum actual_checksum checksum_file

  if [ "$VERIFY_GO_CHECKSUM" = "0" ]; then
    warn "Go checksum verification disabled (VERIFY_GO_CHECKSUM=0)."
    return
  fi

  if [ -n "$GO_CHECKSUM" ]; then
    expected_checksum="$GO_CHECKSUM"
  else
    checksum_file="${tarball_path}.sha256"
    if ! download_file "$checksum_file" "${go_url}.sha256"; then
      err "Failed to download Go checksum file from ${go_url}.sha256"
      exit 1
    fi
    expected_checksum="$(awk '{print $1}' "$checksum_file" | tr -d '\r\n')"
  fi

  if [ -z "$expected_checksum" ]; then
    err "Expected checksum is empty for $go_tarball"
    exit 1
  fi

  actual_checksum="$(calculate_file_sha256 "$tarball_path")"
  if [ "$actual_checksum" != "$expected_checksum" ]; then
    err "Checksum mismatch for $go_tarball"
    err "Expected: $expected_checksum"
    err "Actual:   $actual_checksum"
    exit 1
  fi

  log "Checksum verified for $go_tarball"
}

dry_run_report() {
  local missing=()
  local current_go

  log "Dry run mode enabled. No packages will be installed and no files will be modified."
  log "Resolved settings:"
  log "  SPIDERWEB_REPO=$SPIDERWEB_REPO"
  log "  SPIDERWEB_DIR=$SPIDERWEB_DIR"
  log "  INSTALL_PREFIX=$INSTALL_PREFIX"
  log "  GO_INSTALL_DIR=$GO_INSTALL_DIR"
  log "  MIN_GO_VERSION=$MIN_GO_VERSION"
  log "  GO_VERSION=$GO_VERSION"
  log "  VERIFY_GO_CHECKSUM=$VERIFY_GO_CHECKSUM"

  for cmd in git make tar; do
    if ! have_cmd "$cmd"; then
      missing+=("$cmd")
    fi
  done
  if ! have_cmd curl && ! have_cmd wget; then
    missing+=("curl-or-wget")
  fi

  if [ "${#missing[@]}" -eq 0 ]; then
    log "Prerequisite tools check: OK"
  else
    warn "Missing prerequisites: ${missing[*]}"
    warn "Installer would attempt package installation for these tools."
  fi

  if current_go="$(get_go_version)"; then
    if version_ge "$current_go" "$MIN_GO_VERSION"; then
      log "Go check: OK ($current_go >= $MIN_GO_VERSION)"
    else
      warn "Go check: current version $current_go is below required $MIN_GO_VERSION"
      warn "Installer would install Go $GO_VERSION to $GO_INSTALL_DIR"
    fi
  else
    warn "Go check: not found in PATH"
    warn "Installer would install Go $GO_VERSION to $GO_INSTALL_DIR"
  fi

  if [ "$VERIFY_GO_CHECKSUM" = "0" ]; then
    warn "Checksum verification is disabled (VERIFY_GO_CHECKSUM=0)."
  elif have_cmd sha256sum || have_cmd shasum; then
    log "Checksum verification: enabled and SHA256 tool available"
  else
    warn "Checksum verification is enabled, but sha256sum/shasum is not currently available."
  fi

  if [ -d "$SPIDERWEB_DIR/.git" ]; then
    log "Repository check: existing clone found (would run git pull)"
  elif [ -d "$SPIDERWEB_DIR" ]; then
    warn "Repository check: directory exists but is not a git repo: $SPIDERWEB_DIR"
    warn "Installer would stop and ask for a different SPIDERWEB_DIR."
  else
    log "Repository check: target dir not present (would run git clone)"
  fi

  log "Dry run complete."
}

install_go_local() {
  local os arch go_arch go_tarball go_url tmp_dir tarball_path

  if ! resolve_tar_bin; then
    err "A tar-compatible tool is required but not found (tar/gtar)."
    exit 1
  fi

  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "$arch" in
    x86_64|amd64) go_arch="amd64" ;;
    aarch64|arm64) go_arch="arm64" ;;
    armv6l) go_arch="armv6l" ;;
    armv7l) go_arch="armv6l" ;;
    *)
      err "Unsupported architecture for automatic Go install: $arch"
      exit 1
      ;;
  esac

  go_tarball="go${GO_VERSION}.${os}-${go_arch}.tar.gz"
  go_url="https://go.dev/dl/${go_tarball}"
  tmp_dir="$(mktemp -d)"
  tarball_path="$tmp_dir/$go_tarball"

  log "Installing Go ${GO_VERSION} to ${GO_INSTALL_DIR}"

  download_file "$tarball_path" "$go_url"
  verify_go_tarball_checksum "$go_tarball" "$go_url" "$tarball_path"

  rm -rf "$GO_INSTALL_DIR"
  mkdir -p "$GO_INSTALL_DIR"
  "$TAR_BIN" -C "$GO_INSTALL_DIR" --strip-components=1 -xzf "$tarball_path"
  rm -rf "$tmp_dir"

  export PATH="$GO_INSTALL_DIR/bin:$INSTALL_PREFIX/bin:$PATH"
  ensure_path_exports

  log "Go installed locally at $GO_INSTALL_DIR"
}

ensure_go() {
  local current_go

  if current_go="$(get_go_version)"; then
    if version_ge "$current_go" "$MIN_GO_VERSION"; then
      log "Go $current_go detected (meets minimum $MIN_GO_VERSION)."
      return
    fi

    warn "Go $current_go detected, but $MIN_GO_VERSION+ is required. Upgrading locally."
  else
    warn "Go not found. Installing locally."
  fi

  install_go_local
}

clone_or_update_repo() {
  if [ -d "$SPIDERWEB_DIR/.git" ]; then
    log "Existing spiderweb repository found in $SPIDERWEB_DIR; pulling latest changes."
    git -C "$SPIDERWEB_DIR" pull --ff-only
    return
  fi

  if [ -d "$SPIDERWEB_DIR" ]; then
    err "Directory exists but is not a git repository: $SPIDERWEB_DIR"
    err "Set SPIDERWEB_DIR to an empty/new path and run again."
    exit 1
  fi

  log "Cloning spiderweb repository into $SPIDERWEB_DIR"
  git clone "$SPIDERWEB_REPO" "$SPIDERWEB_DIR"
}

install_spiderweb() {
  export PATH="$GO_INSTALL_DIR/bin:$INSTALL_PREFIX/bin:$PATH"

  log "Running spiderweb build/install steps"
  make -C "$SPIDERWEB_DIR" deps
  make -C "$SPIDERWEB_DIR" build
  make -C "$SPIDERWEB_DIR" install INSTALL_PREFIX="$INSTALL_PREFIX"

  log "Spiderweb installation completed"
}

main() {
  parse_args "$@"

  log "Starting spiderweb installer"
  log "Using installer config: $INSTALL_CONFIG"
  ensure_linux_runtime

  if [ "$DRY_RUN" = "1" ]; then
    dry_run_report
    return 0
  fi

  ensure_basics
  ensure_go
  clone_or_update_repo
  install_spiderweb

  printf '\n'
  log "Done. Ensure this path is available in your shell: $INSTALL_PREFIX/bin"
  log "Binary location: $INSTALL_PREFIX/bin/sweb"
  log "If needed, reload shell config: source ~/.bashrc"
}

main "$@"
