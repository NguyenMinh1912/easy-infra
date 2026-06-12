#!/usr/bin/env sh
# ensure-go.sh — make sure a Go toolchain is available for the build.
#
# If `go` is already on PATH this is a no-op. Otherwise it downloads a pinned
# Go release for the current OS/arch and unpacks it into ./.go so the Makefile
# can build with ./.go/go/bin/go. Invoked by `make install`.
set -eu

# Pinned toolchain version. Keep in sync with the `go` directive in go.mod.
GO_VERSION=1.25.0

# Local bootstrap location (matches LOCAL_GO_BIN in the Makefile).
LOCAL_GO_ROOT="$(CDPATH= cd "$(dirname "$0")/.." && pwd)/.go"
LOCAL_GO="$LOCAL_GO_ROOT/go/bin/go"

# Already have a system Go? Nothing to do.
if command -v go >/dev/null 2>&1; then
	echo "Using system Go: $(command -v go) ($(go version 2>/dev/null))"
	exit 0
fi

# Already bootstrapped a local Go on a previous run? Reuse it.
if [ -x "$LOCAL_GO" ]; then
	echo "Using bootstrapped Go: $LOCAL_GO ($("$LOCAL_GO" version 2>/dev/null))"
	exit 0
fi

echo "No 'go' found on PATH — bootstrapping Go $GO_VERSION into $LOCAL_GO_ROOT ..."

# Detect OS.
os=$(uname -s)
case "$os" in
	Linux)  GOOS=linux ;;
	Darwin) GOOS=darwin ;;
	*)
		echo "ensure-go: unsupported OS '$os'. Please install Go $GO_VERSION manually: https://go.dev/dl/" >&2
		exit 1
		;;
esac

# Detect architecture.
arch=$(uname -m)
case "$arch" in
	x86_64|amd64)        GOARCH=amd64 ;;
	arm64|aarch64)       GOARCH=arm64 ;;
	armv6l|armv7l)       GOARCH=armv6l ;;
	i386|i686)           GOARCH=386 ;;
	*)
		echo "ensure-go: unsupported architecture '$arch'. Please install Go $GO_VERSION manually: https://go.dev/dl/" >&2
		exit 1
		;;
esac

TARBALL="go${GO_VERSION}.${GOOS}-${GOARCH}.tar.gz"
URL="https://go.dev/dl/${TARBALL}"

# Download into a temp dir, extract, then atomically move into place.
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $URL ..."
if command -v curl >/dev/null 2>&1; then
	curl -fSL "$URL" -o "$tmp/$TARBALL"
elif command -v wget >/dev/null 2>&1; then
	wget -O "$tmp/$TARBALL" "$URL"
else
	echo "ensure-go: need 'curl' or 'wget' to download Go. Please install one, or install Go manually: https://go.dev/dl/" >&2
	exit 1
fi

echo "Extracting ..."
tar -C "$tmp" -xzf "$tmp/$TARBALL"

rm -rf "$LOCAL_GO_ROOT"
mkdir -p "$LOCAL_GO_ROOT"
mv "$tmp/go" "$LOCAL_GO_ROOT/go"

if [ ! -x "$LOCAL_GO" ]; then
	echo "ensure-go: bootstrap failed — $LOCAL_GO not found after extract." >&2
	exit 1
fi

echo "Bootstrapped Go: $LOCAL_GO ($("$LOCAL_GO" version 2>/dev/null))"
