#!/usr/bin/env sh

set -eu

DEST_DIR="${DEST_DIR:-$HOME/.local/bin}"
BIN_NAME="vectos"

if ! command -v go >/dev/null 2>&1; then
  printf '%s\n' "Error: Go is not installed or not available in PATH." >&2
  printf '%s\n' "Install Go first: https://go.dev/doc/install" >&2
  exit 1
fi

mkdir -p "$DEST_DIR"

printf '%s\n' "Building vectos..."
go build -o "$BIN_NAME" ./cmd/vectos

printf '%s\n' "Installing to $DEST_DIR/$BIN_NAME..."
install -m 0755 "$BIN_NAME" "$DEST_DIR/$BIN_NAME"

printf '%s\n' "Installed $BIN_NAME to $DEST_DIR/$BIN_NAME"
printf '%s\n' "Make sure $DEST_DIR is in your PATH."
