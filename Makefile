# Argus Makefile
# ===============
# Build and installation commands for the project

# Binary output name
BINARY_NAME=argus

# Build directory
BUILD_DIR=./build

# Go build flags (strip debug info for smaller binary)
GO_BUILD_FLAGS=-ldflags="-s -w" -trimpath

# Installation paths
INSTALL_BIN=/usr/local/bin
CONFIG_DIR=/etc/argus
SUDOERS_DIR=/etc/sudoers.d
GROUP_NAME=argus-users

# Default target
.PHONY: all
all: build

# Build development binary (with debug info)
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) ./cmd/argus
	@echo "Done! Binary: ./$(BINARY_NAME)"

# Build release binary (optimized, stripped)
.PHONY: release
release:
	@echo "Building release $(BINARY_NAME)..."
	go build $(GO_BUILD_FLAGS) -o $(BINARY_NAME) ./cmd/argus
	@echo "Done! Binary: ./$(BINARY_NAME) ($$(du -h $(BINARY_NAME) | cut -f1))"

# Run the application
.PHONY: run
run: build
	./$(BINARY_NAME)

# Run tests
.PHONY: test
test:
	go test -v ./...

# Run tests with race detector
.PHONY: test-race
test-race:
	go test -race -v ./...

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	go clean

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
.PHONY: lint
lint:
	golangci-lint run

# Download dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Show binary size
.PHONY: size
size:
	@du -h $(BINARY_NAME) 2>/dev/null || echo "Binary not built yet"
	@file $(BINARY_NAME) 2>/dev/null || true

# =============================================================================
# Installation Targets (require sudo)
# =============================================================================

# Quick install (binary only, no privilege separation)
.PHONY: install-quick
install-quick: release
	@echo "Quick install to $(INSTALL_BIN)/$(BINARY_NAME)..."
	sudo install -Dm755 $(BINARY_NAME) $(INSTALL_BIN)/$(BINARY_NAME)
	@echo "Done! Run with: sudo argus"

# Full secure install (with wrapper, sudoers, group)
.PHONY: install
install: release
	@echo "Running full installation..."
	sudo bash scripts/install.sh
	@echo "Installation complete!"

# Uninstall
.PHONY: uninstall
uninstall:
	@echo "Uninstalling Argus..."
	sudo rm -f $(INSTALL_BIN)/$(BINARY_NAME)
	sudo rm -f $(INSTALL_BIN)/$(BINARY_NAME)-bin
	sudo rm -f $(SUDOERS_DIR)/argus
	@echo "Uninstalled. Config at $(CONFIG_DIR) preserved."
	@echo "To fully remove: sudo rm -rf $(CONFIG_DIR)"

# Install user config to ~/.config/argus
.PHONY: install-config
install-config:
	@echo "Installing user config..."
	mkdir -p $$HOME/.config/argus
	cp -n configs/default.yaml $$HOME/.config/argus/config.yaml 2>/dev/null || true
	@echo "Config: ~/.config/argus/config.yaml"

# =============================================================================
# Development Targets
# =============================================================================

# Watch for changes and rebuild (requires entr)
.PHONY: watch
watch:
	find . -name "*.go" | entr -c make build

# Generate coverage report
.PHONY: coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# =============================================================================
# Help
# =============================================================================

.PHONY: help
help:
	@echo "Argus Makefile"
	@echo ""
	@echo "Development:"
	@echo "  make build         - Build development binary"
	@echo "  make release       - Build optimized release binary"
	@echo "  make run           - Build and run"
	@echo "  make test          - Run tests"
	@echo "  make fmt           - Format code"
	@echo "  make lint          - Lint code (requires golangci-lint)"
	@echo "  make clean         - Remove build artifacts"
	@echo ""
	@echo "Installation (require sudo):"
	@echo "  make install       - Full install with security features"
	@echo "  make install-quick - Quick install (binary only)"
	@echo "  make install-config- Install user config to ~/.config/argus"
	@echo "  make uninstall     - Remove installed files"
	@echo ""
	@echo "After install, add users to the argus-users group:"
	@echo "  sudo usermod -aG argus-users $$USER"
