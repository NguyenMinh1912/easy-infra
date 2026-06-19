# Makefile for easy-infra.
#
# Common targets:
#   make build     compile the binary into ./easy-infra
#   make install   build the UI bundle, then install easy-infra into $(GOBIN) so it is on your PATH
#   make uninstall remove the installed binary
#   make test      run the test suite
#   make vet       run static checks
#   make fmt       format the code
#   make clean     remove build artifacts

# Binary name and where `go build` drops it.
BINARY := easy-infra
BIN    := ./$(BINARY)

# Local bootstrap location for a Go toolchain auto-installed when the user has
# no system `go` (see scripts/ensure-go.sh, invoked by the `install` target).
LOCAL_GO_BIN := $(CURDIR)/.go/go/bin

# Resolve a usable `go`: the system one if it is on PATH, otherwise the
# bootstrapped copy under ./.go that `make install` downloads on demand.
GO := $(shell command -v go 2>/dev/null || echo $(LOCAL_GO_BIN)/go)

# Install location: prefer GOBIN, then GOPATH/bin. Computed in the recipes
# (via $(GO) env) so it still resolves after a toolchain is bootstrapped.

.DEFAULT_GOAL := build

.PHONY: build install uninstall test vet fmt clean ui ui-install ui-dev dev ensure-go

build:
	$(GO) build -o $(BIN) .

# Ensure a Go toolchain is available: if `go` is not on PATH, download and
# unpack a pinned version into ./.go via scripts/ensure-go.sh.
ensure-go:
	@scripts/ensure-go.sh

# Frontend (ui/, Vite + React + TypeScript).
#   make ui-install  install npm dependencies
#   make ui          build the production bundle into ui/dist (embedded by `serve`)
#   make ui-dev      run the Vite dev server (proxies /api to `easy-infra serve`)
ui-install:
	cd ui && npm install

ui: ui-install
	cd ui && npm run build
	@touch ui/dist/.gitkeep   # vite empties dist/; keep the embed placeholder tracked

ui-dev:
	cd ui && npm run dev

# One-command dev mode with hot reload, no full build: runs the Go backend under
# air (rebuilds/reruns `easy-infra ui` on .go changes) and the Vite dev server
# (HMR) together, tearing both down on Ctrl+C. Open the Vite URL it prints
# (http://localhost:5173). See scripts/dev.sh.
dev:
	@scripts/dev.sh

# Install easy-infra as a global command. Bootstraps a Go toolchain if the
# user has none, builds the UI bundle so the installed binary embeds the
# production frontend, then go install drops the binary in GOBIN; if that
# directory is not on the user's PATH, print the exact line to add it so
# '$(BINARY)' is runnable from anywhere.
install: ensure-go ui
	$(GO) install .
	@GOBIN="$$($(GO) env GOBIN)"; [ -n "$$GOBIN" ] || GOBIN="$$($(GO) env GOPATH)/bin"; \
	echo "Installed $(BINARY) to $$GOBIN"; \
	case ":$$PATH:" in \
		*":$$GOBIN:"*) \
			echo "$$GOBIN is on your PATH — run '$(BINARY)' from anywhere." ;; \
		*) \
			echo "WARNING: $$GOBIN is not on your PATH."; \
			echo "Add it (e.g. to ~/.bashrc), then reload your shell:"; \
			echo "    echo 'export PATH=\"$$GOBIN:\$$PATH\"' >> ~/.bashrc && source ~/.bashrc" ;; \
	esac

uninstall:
	@GOBIN="$$($(GO) env GOBIN)"; [ -n "$$GOBIN" ] || GOBIN="$$($(GO) env GOPATH)/bin"; \
	rm -f "$$GOBIN/$(BINARY)"; \
	echo "Removed $$GOBIN/$(BINARY)"

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

# Format all Go sources, skipping the bootstrapped toolchain under ./.go (which
# ships intentionally-malformed testdata that gofmt would choke on).
fmt:
	@find . -type f -name '*.go' -not -path './.go/*' -print0 | xargs -0 $(dir $(GO))gofmt -w

clean:
	rm -f $(BIN)
	$(GO) clean
