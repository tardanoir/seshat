#!/bin/sh
# seshat installer
#
#   curl -fsSL https://raw.githubusercontent.com/tardanoir/seshat/main/install.sh | sh
#
# Env overrides:
#   SESHAT_VERSION   pin a version (e.g. v1.0.0); defaults to latest release
#   SESHAT_BIN_DIR   install location; defaults to ~/.local/bin (or /usr/local/bin if root)

set -eu

REPO="tardanoir/seshat"
BIN="seshat"

err() { printf 'seshat-install: %s\n' "$1" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

# --- detect platform -------------------------------------------------------
os=$(uname -s)
arch=$(uname -m)

case "$os" in
  Linux)  os=linux ;;
  Darwin) os=darwin ;;
  *) err "unsupported OS: $os (use Scoop on Windows)" ;;
esac

case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) err "unsupported architecture: $arch" ;;
esac

# --- resolve version -------------------------------------------------------
version="${SESHAT_VERSION:-}"
if [ -z "$version" ]; then
  if have curl; then
    version=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
      | grep '"tag_name"' | head -n1 | cut -d '"' -f4)
  elif have wget; then
    version=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" \
      | grep '"tag_name"' | head -n1 | cut -d '"' -f4)
  else
    err "need curl or wget"
  fi
fi
[ -n "$version" ] || err "could not determine latest version"

num="${version#v}"  # strip leading v for the archive name
archive="${BIN}_${num}_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/download/$version/$archive"

# --- choose install dir ----------------------------------------------------
bindir="${SESHAT_BIN_DIR:-}"
if [ -z "$bindir" ]; then
  if [ "$(id -u)" = "0" ]; then
    bindir=/usr/local/bin
  else
    bindir="$HOME/.local/bin"
  fi
fi
mkdir -p "$bindir"

# --- download & install ----------------------------------------------------
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

printf 'Downloading %s...\n' "$archive"
if have curl; then
  curl -fSL "$url" -o "$tmp/$archive" || err "download failed: $url"
else
  wget -qO "$tmp/$archive" "$url" || err "download failed: $url"
fi

tar -xzf "$tmp/$archive" -C "$tmp"
install -m 0755 "$tmp/$BIN" "$bindir/$BIN"

printf '\nInstalled %s %s to %s/%s\n' "$BIN" "$version" "$bindir" "$BIN"
case ":$PATH:" in
  *":$bindir:"*) ;;
  *) printf 'Note: %s is not on your PATH. Add it:\n  export PATH="%s:$PATH"\n' "$bindir" "$bindir" ;;
esac
