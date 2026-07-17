#!/usr/bin/env bash
set -euo pipefail

# Install jira-mcp from GitHub releases, Homebrew, or Go.
# Usage: curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.sh | bash

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
CONFIGURE="${INSTALL_CONFIGURE:-}"
CLIENT="${INSTALL_CLIENT:-}"
JIRA_URL_ARG="${JIRA_URL:-}"
JIRA_USERNAME_ARG="${JIRA_USERNAME:-}"
JIRA_API_TOKEN_ARG="${JIRA_API_TOKEN:-}"
YES="${INSTALL_YES:-}"
INSPECT="${INSTALL_INSPECT:-}"

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
  --configure                 Run MCP client configuration wizard (env: INSTALL_CONFIGURE=1)
  --no-configure              Skip configuration wizard (env: INSTALL_CONFIGURE=0)
  --client <name>             Pre-select MCP client: opencode, claude, cursor, windsurf (env: INSTALL_CLIENT)
  --jira-url <url>            Jira URL (env: JIRA_URL)
  --jira-username <email>     Jira username/email (env: JIRA_USERNAME)
  --jira-api-token <token>    Jira API token (env: JIRA_API_TOKEN)
  --yes, -y                   Skip confirmations (env: INSTALL_YES=1)
  --inspect                   Download and print this script, then exit (env: INSTALL_INSPECT=1)
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
    --configure)
      CONFIGURE="1"
      shift
      ;;
    --no-configure)
      CONFIGURE="0"
      shift
      ;;
    --client)
      [ $# -ge 2 ] || { log_error "--client requires an argument"; exit 1; }
      CLIENT="$2"
      shift 2
      ;;
    --jira-url)
      [ $# -ge 2 ] || { log_error "--jira-url requires an argument"; exit 1; }
      JIRA_URL_ARG="$2"
      shift 2
      ;;
    --jira-username)
      [ $# -ge 2 ] || { log_error "--jira-username requires an argument"; exit 1; }
      JIRA_USERNAME_ARG="$2"
      shift 2
      ;;
    --jira-api-token)
      [ $# -ge 2 ] || { log_error "--jira-api-token requires an argument"; exit 1; }
      JIRA_API_TOKEN_ARG="$2"
      shift 2
      ;;
    --yes|-y)
      YES="1"
      shift
      ;;
    --inspect)
      INSPECT="1"
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

  if has_command brew; then
    JIRA_MCP_BIN="$(brew --prefix)/bin/jira-mcp"
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

  JIRA_MCP_BIN="${dest}"

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

  JIRA_MCP_BIN="${go_bin}/jira-mcp"

  log_success "installed jira-mcp to ${JIRA_MCP_BIN}"

  if ! is_in_path "${go_bin}"; then
    log_warning "${go_bin} is not in your PATH"
    log_info "add it to your shell profile:"
    log_info "  export PATH=\"${go_bin}:\$PATH\""
  fi

  return 0
}

# ---------------------------------------------------------------------------
# Configuration wizard
# ---------------------------------------------------------------------------

is_valid_client() {
  case "$1" in
    opencode|claude|cursor|windsurf) return 0 ;;
    *) return 1 ;;
  esac
}

get_config_path() {
  local client="$1"
  local config_home="${XDG_CONFIG_HOME:-${HOME}/.config}"
  case "${client}" in
    opencode) printf '%s\n' "${config_home}/opencode/opencode.json" ;;
    claude) printf '%s\n' "${config_home}/claude/claude_desktop_config.json" ;;
    cursor) printf '%s\n' "${HOME}/.cursor/mcp.json" ;;
    windsurf) printf '%s\n' "${config_home}/windsurf/mcp_config.json" ;;
  esac
}

prompt_client() {
  if [ -n "${CLIENT}" ]; then
    return 0
  fi
  local input
  printf "Which MCP client do you want to configure? [opencode/claude/cursor/windsurf/none] (default: opencode): "
  if ! read -r input; then
    log_info "EOF detected; skipping configuration"
    CLIENT="none"
    return
  fi
  if [ -z "${input}" ]; then
    input="opencode"
  fi
  case "${input}" in
    opencode|claude|cursor|windsurf) CLIENT="${input}" ;;
    none|skip) CLIENT="none" ;;
    *)
      log_error "invalid client: ${input}"
      prompt_client
      ;;
  esac
}

prompt_jira_url() {
  if [ -n "${JIRA_URL_ARG}" ]; then
    return 0
  fi
  local input
  printf "Jira URL (e.g. https://yourcompany.atlassian.net): "
  if ! read -r input; then
    log_info "EOF detected; skipping configuration"
    return 1
  fi
  if [[ "${input}" != http://* ]] && [[ "${input}" != https://* ]]; then
    log_error "Jira URL must start with http:// or https://"
    prompt_jira_url
  else
    JIRA_URL_ARG="${input}"
  fi
}

prompt_jira_username() {
  if [ -n "${JIRA_USERNAME_ARG}" ]; then
    return 0
  fi
  local input
  printf "Jira username/email: "
  if ! read -r input; then
    log_info "EOF detected; skipping configuration"
    return 1
  fi
  if [[ "${input}" != *@* ]]; then
    log_error "Username must contain '@'"
    prompt_jira_username
  else
    JIRA_USERNAME_ARG="${input}"
  fi
}

prompt_jira_api_token() {
  if [ -n "${JIRA_API_TOKEN_ARG}" ]; then
    return 0
  fi
  local input
  log_info "If you don't have a Jira API token yet, create one at: https://id.atlassian.com/manage-profile/security/api-tokens"
  printf "Jira API token: "
  if ! read -rs input; then
    echo
    log_info "EOF detected; skipping configuration"
    return 1
  fi
  echo
  if [ -z "${input}" ]; then
    log_error "API token is required"
    prompt_jira_api_token
  else
    JIRA_API_TOKEN_ARG="${input}"
  fi
}

confirm_write() {
  local config_path="$1"
  if [ -n "${YES}" ]; then
    return 0
  fi
  local input
  printf "\nWill write jira-mcp configuration for %s at %s\n" "${CLIENT}" "${config_path}"
  printf "Jira URL: %s\n" "${JIRA_URL_ARG}"
  printf "Username: %s\n" "${JIRA_USERNAME_ARG}"
  printf "API token: <hidden>\n"
  printf "Proceed? [Y/n] "
  if ! read -r input; then
    return 1
  fi
  case "${input}" in
    n|N|no|No) return 1 ;;
    *) return 0 ;;
  esac
}

write_config() {
  local client="$1"
  local path="$2"
  local url="$3"
  local username="$4"
  local token="$5"

  local dir
  dir=$(dirname "${path}")
  if ! ensure_dir "${dir}"; then
    log_error "could not create directory: ${dir}"
    exit 1
  fi

  local python_script
  python_script=$(cat <<'PYEOF'
import json, sys, os, shutil

path = sys.argv[1]
client = sys.argv[2]
url = sys.argv[3]
username = sys.argv[4]
token = sys.argv[5]

if client == 'opencode':
    new_entry = {
        'mcp': {
            'jira': {
                'type': 'local',
                'command': ['jira-mcp'],
                'environment': {
                    'JIRA_URL': url,
                    'JIRA_USERNAME': username,
                    'JIRA_API_TOKEN': token
                }
            }
        }
    }
else:
    new_entry = {
        'mcpServers': {
            'jira': {
                'command': 'jira-mcp',
                'env': {
                    'JIRA_URL': url,
                    'JIRA_USERNAME': username,
                    'JIRA_API_TOKEN': token
                }
            }
        }
    }

data = None
file_existed = os.path.exists(path)
if file_existed:
    try:
        with open(path, 'r') as f:
            data = json.load(f)
    except json.JSONDecodeError as e:
        print(f"error: could not parse existing config at {path}: {e}", file=sys.stderr)
        print("Please fix the existing config manually. The new jira entry would be:", file=sys.stderr)
        print(json.dumps(new_entry, indent=2))
        sys.exit(1)

if data is None:
    data = {}

for key, value in new_entry.items():
    if key in data and isinstance(data[key], dict) and isinstance(value, dict):
        data[key].update(value)
    else:
        data[key] = value

if file_existed:
    shutil.copy2(path, path + '.bak')

with open(path, 'w') as f:
    json.dump(data, f, indent=2)
    f.write('\n')
PYEOF
)

  if has_command python3; then
    python3 -c "${python_script}" "${path}" "${client}" "${url}" "${username}" "${token}"
  elif has_command python; then
    python -c "${python_script}" "${path}" "${client}" "${url}" "${username}" "${token}"
  else
    log_error "python3 or python is required to write configuration files"
    exit 1
  fi

  if [ -f "${path}" ]; then
    chmod 600 "${path}" || log_warning "could not restrict permissions on ${path}"
  fi
  if [ -f "${path}.bak" ]; then
    chmod 600 "${path}.bak" || log_warning "could not restrict permissions on ${path}.bak"
  fi
}

configure_client() {
  if [ "${CONFIGURE:-}" = "0" ]; then
    return 0
  fi

  if [ -n "${CLIENT}" ] && { [ "${CLIENT}" = "none" ] || [ "${CLIENT}" = "skip" ]; }; then
    return 0
  fi

  local all_provided=0
  if [ -n "${CLIENT}" ] && [ -n "${JIRA_URL_ARG}" ] && [ -n "${JIRA_USERNAME_ARG}" ] && [ -n "${JIRA_API_TOKEN_ARG}" ]; then
    all_provided=1
  fi

  # Without a TTY we cannot prompt; proceed only if every required value was
  # supplied up front or if the user explicitly forced configuration.
  if [ -z "${CONFIGURE}" ] && [ ! -t 0 ] && [ "${all_provided}" -ne 1 ]; then
    return 0
  fi

  if [ -t 0 ]; then
    print_security_notice
  fi

  if [ -z "${CLIENT}" ]; then
    if [ ! -t 0 ]; then
      return 0
    fi
    prompt_client
  fi

  if [ "${CLIENT}" = "none" ] || [ "${CLIENT}" = "skip" ]; then
    return 0
  fi

  if ! is_valid_client "${CLIENT}"; then
    log_error "invalid client: ${CLIENT}"
    return 1
  fi

  if [ -z "${JIRA_URL_ARG}" ] && [ ! -t 0 ]; then
    return 0
  fi
  prompt_jira_url || return 0

  if [ -z "${JIRA_USERNAME_ARG}" ] && [ ! -t 0 ]; then
    return 0
  fi
  prompt_jira_username || return 0

  if [ -z "${JIRA_API_TOKEN_ARG}" ] && [ ! -t 0 ]; then
    return 0
  fi
  prompt_jira_api_token || return 0

  local config_path
  config_path=$(get_config_path "${CLIENT}")

  # If every value was provided non-interactively, treat that as confirmation
  # because we cannot prompt; otherwise respect --yes or ask.
  if [ "${all_provided}" -ne 1 ] && ! confirm_write "${config_path}"; then
    log_info "configuration skipped"
    return 0
  fi

  write_config "${CLIENT}" "${config_path}" "${JIRA_URL_ARG}" "${JIRA_USERNAME_ARG}" "${JIRA_API_TOKEN_ARG}"

  log_success "Configured jira-mcp for ${CLIENT} at ${config_path}"
  if [ -f "${config_path}.bak" ]; then
    log_info "Backed up previous config to ${config_path}.bak"
  fi
  log_info "You can verify with: jira-mcp --version"
  log_info "Create or verify API tokens: https://id.atlassian.com/manage-profile/security/api-tokens"
}

# ---------------------------------------------------------------------------
# Security notice
# ---------------------------------------------------------------------------

print_security_notice() {
  log_info "Security notes:"
  log_info "  - This script is open source. Inspect it with: --inspect"
  log_info "  - Your Jira API token is stored locally in your MCP client config file."
  log_info "  - The token is never sent anywhere except to your Jira instance."
  log_info "  - Prefer interactive prompts so the token is not saved to shell history."
  log_info "  - Create or verify API tokens: https://id.atlassian.com/manage-profile/security/api-tokens"
}

# ---------------------------------------------------------------------------
# Inspect mode
# ---------------------------------------------------------------------------

inspect_script() {
  local url="https://raw.githubusercontent.com/${REPO}/master/scripts/install.sh"
  if ! has_command curl; then
    log_error "curl is required to download the script for inspection"
    exit 1
  fi
  curl -fsSL "${url}"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
  if [ -n "${INSPECT}" ]; then
    inspect_script
    exit 0
  fi

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

  configure_client

  local version_bin="${JIRA_MCP_BIN:-$(command -v jira-mcp 2>/dev/null || true)}"
  if [ -n "${version_bin}" ] && [ -x "${version_bin}" ]; then
    local installed_version
    installed_version=$("${version_bin}" --version 2>/dev/null || true)
    if [ -n "${installed_version}" ]; then
      log_info "jira-mcp version: ${installed_version}"
    fi
  fi

  log_success "jira-mcp is installed. Run '${JIRA_MCP_BIN:-jira-mcp} --version' to verify."
}

main
