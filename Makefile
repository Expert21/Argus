# Argus Makefile
# ===============
# Common build commands for the project

# GO SYNTAX LESSON: Makefiles aren't Go, but they're essential for project management
# Variables at the top, targets below
# $@ = target name, $< = first dependency

# Binary output name
BINARY_NAME=argus

# Build directory
BUILD_DIR=./build

# Go build flags
# -ldflags="-s -w" strips debug info for smaller binary
# -trimpath removes file system paths from binary
GO_BUILD_FLAGS=-ldflags="-s -w" -trimpath

# Default target (runs when you just type 'make')
.PHONY: all
all: build

# Build the binary (development - with debug info)
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) ./cmd/argus
	@echo "Done! Binary: ./$(BINARY_NAME)"

# Build release binary (smaller, stripped)
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

# Install to /usr/local/bin (requires sudo)
.PHONY: install
install: release
	@echo "Installing to /usr/local/bin/$(BINARY_NAME)-bin..."
	sudo cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)-bin
	sudo chmod 755 /usr/local/bin/$(BINARY_NAME)-bin
	@echo "Installed!"

# Help target
.PHONY: help
help:
	@echo "Argus Makefile targets:"
	@echo "  make build    - Build development binary"
	@echo "  make release  - Build optimized release binary"
	@echo "  make run      - Build and run"
	@echo "  make test     - Run tests"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make fmt      - Format code"
	@echo "  make deps     - Download dependencies"
	@echo "  make install  - Install to /usr/local/bin (sudo)"
	@echo "  make help     - Show this help"
