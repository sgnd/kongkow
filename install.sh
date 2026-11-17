#!/bin/sh
# Install the latest kongkow release.
#
#   curl -fsSL https://raw.githubusercontent.com/sgnd/kongkow/main/install.sh | sh
#
# Installs to /usr/local/bin when writable, otherwise ~/.local/bin.
# Override with KONGKOW_INSTALL_DIR=/some/dir.
# Install one version with KONGKOW_VERSION=v0.1.0.
set -eu

REPO="sgnd/kongkow"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux | darwin) ;;
  mingw* | msys* | cygwin*)
    echo "This installer is for Linux and macOS. On Windows, download" >&2
    echo "kongkow_windows_amd64.zip from https://github.com/$REPO/releases," >&2
    echo "or run this line inside WSL." >&2
    exit 1 ;;
  *) echo "kongkow supports Linux, macOS and Windows; there is no release for $os." >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) echo "There is no kongkow release for this CPU ($(uname -m)): amd64 and arm64 only." >&2; exit 1 ;;
esac
for tool in curl tar; do
  command -v "$tool" >/dev/null 2>&1 || { echo "The installer needs $tool." >&2; exit 1; }
done

if [ -n "${KONGKOW_INSTALL_DIR:-}" ]; then
  dir="$KONGKOW_INSTALL_DIR"
  mkdir -p "$dir" 2>/dev/null || true
  if [ ! -d "$dir" ] || [ ! -w "$dir" ]; then
    echo "KONGKOW_INSTALL_DIR=$dir is not a folder you can write to." >&2
    exit 1
  fi
else
  dir="/usr/local/bin"
  if [ ! -w "$dir" ]; then
    dir="$HOME/.local/bin"
    mkdir -p "$dir"
  fi
fi

file="kongkow_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/latest/download"
if [ -n "${KONGKOW_VERSION:-}" ]; then
  case "$KONGKOW_VERSION" in v*) v=$KONGKOW_VERSION ;; *) v=v$KONGKOW_VERSION ;; esac
  base="https://github.com/$REPO/releases/download/$v"
fi
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

get() {
  if curl -fsSL "$1" -o "$2"; then return 0; fi
  echo "Could not download kongkow. See https://github.com/$REPO/releases" >&2
  exit 1
}

echo "Downloading $file ..."
get "$base/$file" "$tmp/$file"
get "$base/checksums.txt" "$tmp/checksums.txt"

echo "Verifying checksum ..."
expected=$(grep " $file\$" "$tmp/checksums.txt" | cut -d' ' -f1)
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmp/$file" | cut -d' ' -f1)
else
  actual=$(shasum -a 256 "$tmp/$file" | cut -d' ' -f1)
fi
if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
  echo "The download does not match checksums.txt; nothing was installed." >&2
  exit 1
fi

tar --no-same-owner -xzf "$tmp/$file" -C "$tmp"
new="$dir/.kongkow.new.$$"
trap 'rm -rf "$tmp"; rm -f "$new"' EXIT
cp "$tmp/kongkow" "$new"
chmod 755 "$new"
mv -f "$new" "$dir/kongkow"

echo "Installed: $dir/kongkow"
case ":$PATH:" in
  *":$dir:"*) ;;
  *)
    echo
    echo "Add $dir to PATH, then run: kongkow version"
    ;;
esac
echo "Next: kongkow guide"
