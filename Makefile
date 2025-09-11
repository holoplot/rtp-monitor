.PHONY: build run run-ui clean test fmt fmt-fix vet deps help build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 release

# Build variables
BINARY_NAME=rtp-monitor
BUILD_DIR=./bin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")
GIT_COMMIT?=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC' 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X 'github.com/holoplot/rtp-monitor/internal/version.Version=$(VERSION)' -X 'github.com/holoplot/rtp-monitor/internal/version.GitCommit=$(GIT_COMMIT)' -X 'github.com/holoplot/rtp-monitor/internal/version.BuildDate=$(BUILD_DATE)'"

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Built $(BINARY_NAME) successfully"

# Build and install to system
install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) .
	@echo "Installed $(BINARY_NAME) to $(shell go env GOPATH)/bin/"

# Run the application
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code (check only)
fmt:
	@echo "Checking code formatting..."
	@if [ "$(shell gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "The following files are not formatted:"; \
		gofmt -s -l .; \
		exit 1; \
	fi
	@echo "All files are properly formatted"

# Format code (apply fixes)
fmt-fix:
	@echo "Formatting code..."
	@go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	@go vet ./...

# Lint code (requires golangci-lint to be installed)
lint:
	@echo "Linting code..."
	@golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean

# Development build (with race detector)
dev-build:
	@echo "Building development version..."
	@mkdir -p $(BUILD_DIR)
	@go build -race $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-dev .

# Individual platform build targets for CI
build-linux-amd64:
	@echo "Building for Linux AMD64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

build-linux-arm64:
	@echo "Building for Linux ARM64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

build-darwin-amd64:
	@echo "Building for macOS AMD64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

build-darwin-arm64:
	@echo "Building for macOS ARM64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

build-windows-amd64:
	@echo "Building for Windows AMD64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).exe .

# Create release builds for multiple platforms
release:
	@echo "Creating release builds for version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)/release
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-$(VERSION)-linux-amd64 .
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-$(VERSION)-linux-arm64 .
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-$(VERSION)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-$(VERSION)-darwin-arm64 .
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-$(VERSION)-windows-amd64.exe .
	@cd $(BUILD_DIR)/release && \
		tar -czf $(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-$(VERSION)-linux-amd64 && \
		tar -czf $(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-$(VERSION)-linux-arm64 && \
		tar -czf $(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-$(VERSION)-darwin-amd64 && \
		tar -czf $(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-$(VERSION)-darwin-arm64 && \
		zip $(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-$(VERSION)-windows-amd64.exe && \
		rm $(BINARY_NAME)-$(VERSION)-linux-amd64 $(BINARY_NAME)-$(VERSION)-linux-arm64 $(BINARY_NAME)-$(VERSION)-darwin-amd64 $(BINARY_NAME)-$(VERSION)-darwin-arm64 $(BINARY_NAME)-$(VERSION)-windows-amd64.exe && \
		sha256sum $(BINARY_NAME)-$(VERSION)-*.tar.gz $(BINARY_NAME)-$(VERSION)-*.zip > checksums.txt
	@echo "Release builds created in $(BUILD_DIR)/release/ for version $(VERSION)"

# Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# Check for security vulnerabilities
security:
	@echo "Checking for security vulnerabilities..."
	@go list -json -m all | nancy sleuth

# Generate documentation
docs:
	@echo "Generating documentation..."
	@go doc -all ./... > docs.txt
	@echo "Documentation generated: docs.txt"

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  install      - Build and install to system"
	@echo "  run          - Build and run the application"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  fmt          - Check code formatting"
	@echo "  fmt-fix      - Format code (apply fixes)"
	@echo "  vet          - Vet code"
	@echo "  lint         - Lint code (requires golangci-lint)"
	@echo "  clean        - Clean build artifacts"
	@echo "  dev-build    - Build development version with race detector"
	@echo "  release      - Create release builds for multiple platforms"
	@echo "  version      - Show version information"
	@echo "  build-linux-amd64   - Build for Linux AMD64"
	@echo "  build-linux-arm64   - Build for Linux ARM64"
	@echo "  build-darwin-amd64  - Build for macOS AMD64"
	@echo "  build-darwin-arm64  - Build for macOS ARM64"
	@echo "  build-windows-amd64 - Build for Windows AMD64"
	@echo "  security     - Check for security vulnerabilities"
	@echo "  docs         - Generate documentation"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Build Variables:"
	@echo "  VERSION      - Version from git describe (auto-detected: $(VERSION))"
	@echo "  GIT_COMMIT   - Git commit hash (auto-detected)"
	@echo "  BUILD_DATE   - Build timestamp (auto-generated)"
