#!/usr/bin/env sh
# Vectos installer
#
# Downloads the correct experimental release asset for the current platform,
# verifies the checksum, and installs the vectos binary.
#
# Usage (from a release):
#   curl -fsSL https://github.com/mddiosc/vectos/releases/latest/download/install.sh | sh
#
# Or with a custom install directory:
#   curl -fsSL .../install.sh | DEST_DIR=/usr/local/bin sh
#
# Source-based fallback (requires Go):
#   ./scripts/install.sh --from-source

set -eu

REPO="mddiosc/vectos"
BIN_NAME="vectos"
DEST_DIR="${DEST_DIR:-$HOME/.local/bin}"

# ── helpers ──────────────────────────────────────────────────────────────────

die() { printf 'error: %s\n' "$*" >&2; exit 1; }
info() { printf '  %s\n' "$*"; }
ok()   { printf '✓ %s\n' "$*"; }

need() {
  command -v "$1" >/dev/null 2>&1 || die "required tool not found: $1"
}

# ── source-based fallback ────────────────────────────────────────────────────

install_from_source() {
  info "Installing vectos from source (Go required)..."
  need go
  mkdir -p "$DEST_DIR"
  go build -o "$BIN_NAME" ./cmd/vectos
  install -m 0755 "$BIN_NAME" "$DEST_DIR/$BIN_NAME"
  rm -f "$BIN_NAME"
  ok "Installed vectos to $DEST_DIR/$BIN_NAME"
  check_path
  exit 0
}

if [ "${1:-}" = "--from-source" ]; then
  install_from_source
fi

# ── uninstall ─────────────────────────────────────────────────────────────────

uninstall() {
  BIN_PATH="${DEST_DIR}/${BIN_NAME}"

  if [ ! -f "$BIN_PATH" ]; then
    printf 'Nothing to remove: %s not found.\n' "$BIN_PATH"
    printf 'If you installed to a custom directory, set DEST_DIR and retry:\n'
    printf '  DEST_DIR=/usr/local/bin %s --uninstall\n' "$0"
    exit 1
  fi

  rm -f "$BIN_PATH"
  ok "Removed $BIN_PATH"

  printf '\n'
  printf 'Optional manual cleanup:\n'
  printf '  Cached models and index data:  rm -rf ~/.vectos/\n'
  printf '  OpenCode MCP config:           Edit ~/.config/opencode/opencode.json\n'
  printf '  OpenCode global guidance:      Edit ~/.config/opencode/AGENTS.md\n'
  exit 0
}

if [ "${1:-}" = "--uninstall" ]; then
  uninstall
fi

# ── platform detection ────────────────────────────────────────────────────────

detect_os() {
  case "$(uname -s)" in
    Darwin) printf 'darwin' ;;
    Linux)  printf 'linux'  ;;
    *)      die "unsupported OS: $(uname -s). Use --from-source to build manually." ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    arm64|aarch64) printf 'arm64' ;;
    x86_64|amd64)  printf 'amd64' ;;
    *)             die "unsupported architecture: $(uname -m). Use --from-source to build manually." ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

# ── validate platform ─────────────────────────────────────────────────────────

SUPPORTED=0
case "${OS}/${ARCH}" in
  darwin/arm64|linux/amd64) SUPPORTED=1 ;;
esac

if [ "$SUPPORTED" = "0" ]; then
  printf 'Platform %s/%s is not supported by experimental release assets.\n' "$OS" "$ARCH"
  printf 'To install from source instead, run:\n'
  printf '  ./scripts/install.sh --from-source\n'
  exit 1
fi

# ── resolve version ───────────────────────────────────────────────────────────

need curl

if [ -z "${VERSION:-}" ]; then
  info "Detecting latest release..."
  # GitHub API: latest prerelease. Falls back to the first release in the list.
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases" \
    | grep '"tag_name"' \
    | head -1 \
    | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
  [ -n "$VERSION" ] || die "could not determine latest release version"
fi

info "Installing vectos ${VERSION} for ${OS}/${ARCH}..."

# ── download ──────────────────────────────────────────────────────────────────

BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
ARCHIVE="vectos_${VERSION}_${OS}_${ARCH}.tar.gz"
CHECKSUMS="checksums.txt"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

info "Downloading ${ARCHIVE}..."
curl -fsSL "${BASE_URL}/${ARCHIVE}" -o "${TMP}/${ARCHIVE}" \
  || die "failed to download ${BASE_URL}/${ARCHIVE}"

info "Downloading checksums..."
curl -fsSL "${BASE_URL}/${CHECKSUMS}" -o "${TMP}/${CHECKSUMS}" \
  || die "failed to download ${BASE_URL}/${CHECKSUMS}"

# ── verify checksum ───────────────────────────────────────────────────────────

info "Verifying checksum..."
cd "$TMP"

if command -v sha256sum >/dev/null 2>&1; then
  grep "${ARCHIVE}" "${CHECKSUMS}" | sha256sum -c - >/dev/null 2>&1 \
    || die "checksum verification failed for ${ARCHIVE}"
elif command -v shasum >/dev/null 2>&1; then
  grep "${ARCHIVE}" "${CHECKSUMS}" | shasum -a 256 -c - >/dev/null 2>&1 \
    || die "checksum verification failed for ${ARCHIVE}"
else
  info "Warning: no sha256sum or shasum found — skipping checksum verification"
fi

ok "Checksum verified"

# ── install ───────────────────────────────────────────────────────────────────

info "Extracting..."
tar -xzf "${ARCHIVE}"

mkdir -p "$DEST_DIR"
install -m 0755 "${BIN_NAME}" "${DEST_DIR}/${BIN_NAME}"

ok "Installed vectos ${VERSION} to ${DEST_DIR}/${BIN_NAME}"

# ── path reminder ─────────────────────────────────────────────────────────────

check_path() {
  case ":${PATH}:" in
    *":${DEST_DIR}:"*) ;;
    *)
      printf '\n'
      printf '  ⚠️  %s is not in your PATH.\n' "$DEST_DIR"
      printf '  Add it with:\n'
      printf '    export PATH="%s:$PATH"\n' "$DEST_DIR"
      printf '  Or add that line to your ~/.bashrc / ~/.zshrc\n'
      printf '\n'
      ;;
  esac
}

check_path

printf '\nRun `vectos version` to verify the installation.\n'
printf '\n'
printf '  ⚠️  Experimental/internal build — not a stable public release.\n'
printf '  The embedded provider downloads ONNX Runtime and model assets\n'
printf '  on first use into ~/.vectos/models/\n'
