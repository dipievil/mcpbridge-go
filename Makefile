.PHONY: all build clean help install test e2e test-all fmt lint

GO := go
GOOS := linux
GOARCH := amd64
BINARY_NAME := mcpbridgego
BIN_DIR := bin
MAIN_PATH := ./cmd/mcpbridgego
VERSION ?= $(shell git describe --tags 2>/dev/null || echo "dev")

LDFLAGS := -ldflags "-X main.buildVersion=$(VERSION)"

all: build

## build: Compile binary to bin/mcpbridgego
build: | $(BIN_DIR)
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	chmod +x $(BIN_DIR)/$(BINARY_NAME)
	@echo "✓ Build complete: $(BIN_DIR)/$(BINARY_NAME)"

## test: Run unit tests with coverage
test:
	$(GO) test -v -cover ./...
	@echo "✓ Unit tests passed"

## e2e: Run end-to-end tests
e2e: build
	cd tests && $(GO) test -v -tags=e2e -run TestE2E
	@echo "✓ E2E tests passed"

## test-all: Run all tests (unit + e2e)
test-all: test e2e
	@echo "✓ All tests passed"

## fmt: Format code with gofmt
fmt:
	$(GO) fmt ./...
	@echo "✓ Code formatted"

## lint: Run go vet linter
lint:
	$(GO) vet ./...
	@echo "✓ Lint passed"

## clean: Clean build artifacts
clean:
	rm -rf $(BIN_DIR)
	$(GO) clean
	@echo "✓ Clean complete"

## help: Display available targets
help:
	@echo "Available targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)
