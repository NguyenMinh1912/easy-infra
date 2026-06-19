#!/usr/bin/env sh
# dev.sh — run easy-infra in dev mode with hot reload, no full build.
#
# Starts two processes side by side and tears both down on Ctrl+C:
#   1. air     — rebuilds & reruns `easy-infra ui` (JSON API on :8080) whenever a
#                .go file changes (config in .air.toml).
#   2. Vite    — the frontend dev server (HMR) on :5173, proxying /api to :8080
#                (see ui/vite.config.ts). Open this URL in your browser.
#
# Invoked by `make dev`.
set -eu

# Repo root, regardless of where this script is called from.
ROOT="$(CDPATH= cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Resolve a usable `go`: the system one if on PATH, else the copy bootstrapped
# under ./.go by `make install` (mirrors the Makefile's GO resolution). Put its
# bin dir on PATH so air's own `go build` uses the same toolchain.
LOCAL_GO="$ROOT/.go/go/bin/go"
if command -v go >/dev/null 2>&1; then
	GO="$(command -v go)"
elif [ -x "$LOCAL_GO" ]; then
	GO="$LOCAL_GO"
else
	echo "dev: no 'go' on PATH and no bootstrapped toolchain at $LOCAL_GO." >&2
	echo "     Install Go, or run 'make install' once to bootstrap one." >&2
	exit 1
fi
PATH="$(dirname "$GO"):$PATH"
export PATH

# `//go:embed all:dist` (ui/embed.go) needs ui/dist to exist with >=1 file or the
# Go build fails. In dev the live UI is served by Vite, not the embedded bundle,
# so a placeholder is enough to let air's rebuilds compile.
if [ ! -e ui/dist/.gitkeep ]; then
	mkdir -p ui/dist
	touch ui/dist/.gitkeep
fi

# Ensure frontend deps are present before starting Vite.
if [ ! -d ui/node_modules ]; then
	echo "dev: installing frontend dependencies (ui/node_modules missing) ..."
	(cd ui && npm install)
fi

# Track child PIDs and kill both on exit/interrupt. Modern npm and air forward
# termination to their own children (vite, the built binary), so signalling
# these two PIDs tears the whole tree down — no orphaned vite/easy-infra.
pids=""
cleanup() {
	trap - INT TERM EXIT
	[ -n "$pids" ] && kill $pids 2>/dev/null || true
	wait 2>/dev/null || true
}
trap cleanup INT TERM EXIT

echo "dev: starting Vite dev server (:5173) and air-watched backend (:8080) ..."
echo "dev: open http://localhost:5173  —  press Ctrl+C to stop both."

# Use exec in each subshell so $! is the real process (no wrapper sh layer).
(cd ui && exec npm run dev) &
pids="$pids $!"

(exec "$GO" tool air -c .air.toml) &
pids="$pids $!"

# Wait for both; Ctrl+C triggers the trap above.
wait
