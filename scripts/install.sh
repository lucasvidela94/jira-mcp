#!/usr/bin/env bash
set -euo pipefail

# Install jira-mcp from GitHub releases, Homebrew, or Go.
# Usage: curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/main/scripts/install.sh | bash

REPO="lucasvidela94/jira-mcp"
TAP="lucasvidela94/tap"
TMP_DIR=""

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------

cleanup() {
  if [ -n "${TMP_DIR}" ] && [ -d "${TMP_DIR}" ]; then
    rm -rf "${TMP_DIR}"
  fi
}

trap cleanup EXIT

# ---------------------------------------------------------------------------
# Colors and logging
# ---------------------------------------------------------------------------

NO_COLOR="${NO_COLOR:-}"
CI="${CI:-}"
if [ -n "${NO_COLOR}" ] || [ -n "${CI}" ]; then
  RED=""
  GREEN=""
  YELLOW=""
  BLUE=""
  BOLD=""
  RESET=""
else
  RED="\033[31m"
  GREEN="\033[32m"
  YELLOW="\033[33m"
  BLUE="\033[34m"
  BOLD="\033[1m"
  RESET="\033[0m"
fi

log_info() {
  printf "${BLUE}info:${RESET} %s\n" "$*" >&2
}

log_success() {
  printf "${GREEN}success:${RESET} %s\n" "$*" >&2
}

log_error() {
  printf "${RED}error:${RESET} %s\n" "$*" >&2
}

log_warning() {
  printf "${YELLOW}warning:${RESET} %s\n" "$*" >&2
}

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------

METHOD="${INSTALL_METHOD:-}"
INSTALL_DIR="${INSTALL_DIR:-}"
VERSION="${INSTALL_VERSION:-}"
INSECURE="${INSTALL_INSECURE:-}"
SUDO="${INSTALL_SUDO:-}"

usage() {
  cat <<EOF
Install jira-mcp from GitHub releases, Homebrew, or Go.

Usage: $0 [OPTIONS]

Options:
  --method <brew|binary|go>   Install method (env: INSTALL_METHOD)
  --dir <dir>                 Installation directory (env: INSTALL_DIR)
  --version <version>         Version to install, e.g. v1.2.3 (env: INSTALL_VERSION)
  --insecure                  Skip checksum verification (env: INSTALL_INSECURE=1)
  --sudo                      Use sudo for system directories (env: INSTALL_SUDO=1)
  --help, -h                  Show this help message

Defaults:
  method:   auto (brew, binary, go)
  dir:      /usr/local/bin if writable, otherwise \$HOME/.local/bin
  version:  latest
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --method)
      [ $# -ge 2 ] || { log_error "--method requires an argument"; exit 1; }
      METHOD="$2"
      shift 2
      ;;
    --dir)
      [ $# -ge 2 ] || { log_error "--dir requires an argument"; exit 1; }
      INSTALL_DIR="$2"
      shift 2
      ;;
    --version)
      [ $# -ge 2 ] || { log_error "--version requires an argument"; exit 1; }
      VERSION="$2"
      shift 2
      ;;
    --insecure)
      INSECURE="1"
      shift
      ;;
    --sudo)
      SUDO="1"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      log_error "unknown argument: $1"
      usage
      exit 1
      ;;
  esac
done

# ---------------------------------------------------------------------------
# OS / ARCH detection
# ---------------------------------------------------------------------------

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) log_error "unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  msys*|cygwin*|mingw*|nt|win*) OS="windows" ;;
  *) log_error "unsupported OS: $OS"; exit 1 ;;
esac

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

has_command() {
  command -v "$1" >/dev/null 2>&1
}

ensure_dir() {
  local dir="$1"
  if [ ! -d "$dir" ]; then
    mkdir -p "$dir" || return 1
  fi
  return 0
}

is_in_path() {
  local dir="$1"
  case ":${PATH}:" in
    *":${dir}:"*) return 0 ;;
    *) return 1 ;;
  esac
}

# ---------------------------------------------------------------------------
# Version resolution
# ---------------------------------------------------------------------------

resolve_version() {
  if [ -n "${VERSION}" ]; then
    printf '%s\n' "${VERSION}"
    return 0
  fi

  if ! has_command curl; then
    log_error "curl is required to download releases"
    exit 1
  fi

  local tag
  tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -z "${tag}" ]; then
    log_error "could not determine latest version from GitHub"
    exit 1
  fi
  printf '%s\n' "${tag}"
}

# ---------------------------------------------------------------------------
# Install methods
# ---------------------------------------------------------------------------

install_brew() {
  if ! has_command brew; then
    log_info "brew not found; trying binary install"
    return 1
  fi

  log_info "attempting install via Homebrew..."

  if brew tap | grep -q "^${TAP}\$"; then
    : # tap already present
  else
    log_info "tapping ${TAP}..."
    brew tap "${TAP}" || return 1
  fi

  if [ -n "${VERSION}" ]; then
    brew install "${TAP}/jira-mcp" || return 1
  else
    brew update && brew install jira-mcp || return 1
  fi

  log_success "installed jira-mcp via Homebrew"
  return 0
}

install_binary() {
  if ! has_command curl; then
    log_error "curl is required to download releases"
    exit 1
  fi

  local version
  version=$(resolve_version)

  local version_no_v="${version#v}"
  local archive
  local checksum_file="checksums.txt"
  local binary="jira-mcp"
  local ext="tar.gz"

  if [ "$OS" = "windows" ]; then
    ext="zip"
    binary="jira-mcp.exe"
  fi

  archive="jira-mcp_${version_no_v}_${OS}_${ARCH}.${ext}"

  local base_url="https://github.com/${REPO}/releases/download/${version}"
  local archive_url="${base_url}/${archive}"
  local checksum_url="${base_url}/${checksum_file}"

  log_info "installing jira-mcp ${version} for ${OS}/${ARCH}..."

  TMP_DIR=$(mktemp -d)

  curl -fsSL "${archive_url}" -o "${TMP_DIR}/${archive}"

  if [ -z "${INSECURE}" ]; then
    log_info "verifying checksum..."
    if ! curl -fsSL "${checksum_url}" -o "${TMP_DIR}/${checksum_file}"; then
      log_error "could not download checksums.txt"
      exit 1
    fi

    local expected
    expected=$(grep "${archive}" "${TMP_DIR}/${checksum_file}" | awk '{print $1}')
    if [ -z "${expected}" ]; then
      log_error "checksum not found in ${checksum_file}"
      exit 1
    fi

    local actual
    if has_command sha256sum; then
      actual=$(sha256sum "${TMP_DIR}/${archive}" | awk '{print $1}')
    elif has_command shasum; then
      actual=$(shasum -a 256 "${TMP_DIR}/${archive}" | awk '{print $1}')
    else
      log_error "sha256sum or shasum is required to verify checksums"
      exit 1
    fi

    if [ "${actual}" != "${expected}" ]; then
      log_error "checksum mismatch for ${archive}"
      log_error "  expected: ${expected}"
      log_error "  actual:   ${actual}"
      exit 1
    fi
    log_info "checksum verified"
  else
    log_warning "skipping checksum verification (--insecure)"
  fi

  case "${archive}" in
    *.zip) unzip -q "${TMP_DIR}/${archive}" -d "${TMP_DIR}" ;;
    *) tar -xzf "${TMP_DIR}/${archive}" -C "${TMP_DIR}" ;;
  esac

  if [ ! -f "${TMP_DIR}/${binary}" ]; then
    log_error "extracted archive does not contain ${binary}"
    exit 1
  fi

  local install_dir
  if [ -n "${INSTALL_DIR}" ]; then
    install_dir="${INSTALL_DIR}"
  elif [ -w "/usr/local/bin" ]; then
    install_dir="/usr/local/bin"
  else
    install_dir="${HOME}/.local/bin"
  fi

  if ! ensure_dir "${install_dir}"; then
    if [ "${install_dir}" = "/usr/local/bin" ] && [ -z "${SUDO}" ]; then
      log_info "${install_dir} is not writable; retrying with sudo..."
      SUDO="1"
      if ! ensure_dir "${install_dir}"; then
        log_error "could not create directory: ${install_dir}"
        exit 1
      fi
    else
      log_error "could not create directory: ${install_dir}"
      exit 1
    fi
  fi

  if [ -z "${SUDO}" ] && [ ! -w "${install_dir}" ]; then
    if [ "${install_dir}" = "/usr/local/bin" ]; then
      log_info "${install_dir} is not writable; retrying with sudo..."
      SUDO="1"
    else
      log_error "directory is not writable: ${install_dir}"
      exit 1
    fi
  fi

  local dest="${install_dir}/${binary}"
  if [ -n "${SUDO}" ]; then
    if ! has_command sudo; then
      log_error "sudo is required to install to ${install_dir}"
      exit 1
    fi
    sudo cp "${TMP_DIR}/${binary}" "${dest}"
    sudo chmod +x "${dest}"
  else
    cp "${TMP_DIR}/${binary}" "${dest}"
    chmod +x "${dest}"
  fi

  log_success "installed ${binary} to ${dest}"

  if ! is_in_path "${install_dir}"; then
    log_warning "${install_dir} is not in your PATH"
    log_info "add it to your shell profile:"
    log_info "  export PATH=\"${install_dir}:\$PATH\""
  fi

  return 0
}

install_go() {
  if ! has_command go; then
    log_error "go is required for method=go"
    exit 1
  fi

  local version
  version=$(resolve_version)

  log_info "installing jira-mcp ${version} via go install..."
  go install "github.com/${REPO}@${version}"

  local go_bin
  go_bin=$(go env GOBIN)
  if [ -z "${go_bin}" ]; then
    go_bin=$(go env GOPATH)/bin
  fi

  log_success "installed jira-mcp to ${go_bin}/jira-mcp"

  if ! is_in_path "${go_bin}"; then
    log_warning "${go_bin} is not in your PATH"
    log_info "add it to your shell profile:"
    log_info "  export PATH=\"${go_bin}:\$PATH\""
  fi

  return 0
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
  case "${METHOD}" in
    brew)
      install_brew
      ;;
    binary)
      install_binary
      ;;
    go)
      install_go
      ;;
    "")
      if ! install_brew; then
        install_binary
      fi
      ;;
    *)
      log_error "unknown install method: ${METHOD}"
      usage
      exit 1
      ;;
  esac

  if command -v jira-mcp >/dev/null 2>&1; then
    local installed_version
    installed_version=$(jira-mcp --version 2>/dev/null || true)
    if [ -n "${installed_version}" ]; then
      log_info "jira-mcp version: ${installed_version}"
    fi
  fi

  log_success "jira-mcp is installed. Run 'jira-mcp --version' to verify."
}

main
