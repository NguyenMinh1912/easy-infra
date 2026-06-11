# Makefile for easy-infra.
#
# Common targets:
#   make build     compile the binary into ./easy-infra
#   make install   install easy-infra into $(GOBIN) so it is on your PATH
#   make uninstall remove the installed binary
#   make test      run the test suite
#   make vet       run static checks
#   make fmt       format the code
#   make clean     remove build artifacts

# Binary name and where `go build` drops it.
BINARY := easy-infra
BIN    := ./$(BINARY)

# Install location: prefer GOBIN, then GOPATH/bin, then ~/go/bin.
GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif

.DEFAULT_GOAL := build

.PHONY: build install uninstall test vet fmt clean

build:
	go build -o $(BIN) .

# Install easy-infra as a global command. go install drops the binary in
# $(GOBIN); if that directory is not on the user's PATH, print the exact
# line to add it so '$(BINARY)' is runnable from anywhere.
install:
	go install .
	@echo "Installed $(BINARY) to $(GOBIN)"
	@case ":$$PATH:" in \
		*":$(GOBIN):"*) \
			echo "$(GOBIN) is on your PATH — run '$(BINARY)' from anywhere." ;; \
		*) \
			echo "WARNING: $(GOBIN) is not on your PATH."; \
			echo "Add it (e.g. to ~/.bashrc), then reload your shell:"; \
			echo "    echo 'export PATH=\"$(GOBIN):\$$PATH\"' >> ~/.bashrc && source ~/.bashrc" ;; \
	esac

uninstall:
	rm -f $(GOBIN)/$(BINARY)
	@echo "Removed $(GOBIN)/$(BINARY)"

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -f $(BIN)
	go clean
