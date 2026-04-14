.PHONY: all build clean help install test

GO := go
GOOS := linux
GOARCH := amd64
BINARY_NAME := mcpbridgego
BIN_DIR := bin
MAIN_PATH := ./cmd/mcpbridgego
VERSION ?= dev
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

all: build

build: | $(BIN_DIR)
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	chmod +x $(BIN_DIR)/$(BINARY_NAME)
	@echo "✓ Build complete: $(BIN_DIR)/$(BINARY_NAME)"

clean:
	rm -rf $(BIN_DIR)
	$(GO) clean
	@echo "✓ Clean complete"

help:
	@echo "Available targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)
