#!/usr/bin/env sh
# install.sh — install easy-infra with a single command.
#
#   curl -fsSL https://raw.githubusercontent.com/NguyenMinh1912/easy-infra/main/install.sh | sh
#
# Downloads the prebuilt binary matching your OS/arch from the latest GitHub
# Release (published by .github/workflows/release.yml), verifies its checksum,
# and installs it onto your PATH. No Go toolchain required.
#
# Environment overrides:
#   EASY_INFRA_VERSION  install a specific tag (e.g. v1.2.3) instead of latest
#   EASY_INFRA_BIN_DIR  install directory (default: /usr/local/bin, falling back
#                       to $HOME/.local/bin when /usr/local/bin is not writable)
#   EASY_INFRA_REPO     owner/repo to download from (default below)
set -eu

REPO="${EASY_INFRA_REPO:-NguyenMinh1912/easy-infra}"
BINARY="easy-infra"

err() {
	echo "install: $*" >&2
	exit 1
}

# Resolve a downloader once; reuse for every fetch.
if command -v curl >/dev/null 2>&1; then
	dl() { curl -fsSL "$1"; }
	dl_to() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
	dl() { wget -qO- "$1"; }
	dl_to() { wget -qO "$2" "$1"; }
else
	err "need 'curl' or 'wget' to download easy-infra"
fi

# Detect OS. Keep in sync with the matrix in .github/workflows/release.yml.
os=$(uname -s)
case "$os" in
	Linux) GOOS=linux ;;
	Darwin) GOOS=darwin ;;
	*) err "unsupported OS '$os' — see https://github.com/$REPO/releases for manual install" ;;
esac

# Detect architecture.
arch=$(uname -m)
case "$arch" in
	x86_64 | amd64) GOARCH=amd64 ;;
	arm64 | aarch64) GOARCH=arm64 ;;
	*) err "unsupported architecture '$arch' — see https://github.com/$REPO/releases for manual install" ;;
esac

# Resolve the version to install (latest unless pinned).
VERSION="${EASY_INFRA_VERSION:-}"
if [ -z "$VERSION" ]; then
	echo "Resolving latest release of $REPO ..."
	VERSION=$(dl "https://api.github.com/repos/$REPO/releases/latest" \
		| grep '"tag_name"' | head -n1 | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')
	[ -n "$VERSION" ] || err "could not determine latest release — set EASY_INFRA_VERSION to a tag"
fi

ASSET="${BINARY}_${GOOS}_${GOARCH}.tar.gz"
BASE_URL="https://github.com/$REPO/releases/download/$VERSION"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $ASSET ($VERSION) ..."
dl_to "$BASE_URL/$ASSET" "$tmp/$ASSET" || err "download failed: $BASE_URL/$ASSET"

# Verify the checksum when the release publishes one.
if dl_to "$BASE_URL/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
	expected=$(grep " $ASSET\$" "$tmp/checksums.txt" | awk '{print $1}')
	if [ -n "$expected" ]; then
		if command -v sha256sum >/dev/null 2>&1; then
			actual=$(sha256sum "$tmp/$ASSET" | awk '{print $1}')
		elif command -v shasum >/dev/null 2>&1; then
			actual=$(shasum -a 256 "$tmp/$ASSET" | awk '{print $1}')
		fi
		if [ -n "${actual:-}" ] && [ "$actual" != "$expected" ]; then
			err "checksum mismatch for $ASSET (expected $expected, got $actual)"
		fi
		[ -n "${actual:-}" ] && echo "Checksum OK."
	fi
fi

echo "Extracting ..."
tar -C "$tmp" -xzf "$tmp/$ASSET"
[ -f "$tmp/$BINARY" ] || err "archive did not contain '$BINARY'"
chmod +x "$tmp/$BINARY"

# Pick an install directory: explicit override, else /usr/local/bin if we can
# write there, else ~/.local/bin.
BIN_DIR="${EASY_INFRA_BIN_DIR:-}"
if [ -z "$BIN_DIR" ]; then
	if [ -w /usr/local/bin ] 2>/dev/null; then
		BIN_DIR=/usr/local/bin
	else
		BIN_DIR="$HOME/.local/bin"
	fi
fi
mkdir -p "$BIN_DIR"

if mv "$tmp/$BINARY" "$BIN_DIR/$BINARY" 2>/dev/null; then
	:
elif command -v sudo >/dev/null 2>&1; then
	echo "Installing to $BIN_DIR requires elevated permissions ..."
	sudo mv "$tmp/$BINARY" "$BIN_DIR/$BINARY"
else
	err "cannot write to $BIN_DIR — set EASY_INFRA_BIN_DIR to a writable directory"
fi

echo "Installed $BINARY $VERSION to $BIN_DIR/$BINARY"

# Nudge the user if the install dir is not on PATH.
case ":$PATH:" in
	*":$BIN_DIR:"*) echo "Run '$BINARY --help' to get started." ;;
	*)
		echo "WARNING: $BIN_DIR is not on your PATH."
		echo "Add it (e.g. to ~/.bashrc), then reload your shell:"
		echo "    echo 'export PATH=\"$BIN_DIR:\$PATH\"' >> ~/.bashrc && source ~/.bashrc"
		;;
esac
