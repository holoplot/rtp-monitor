.PHONY: build run run-ui clean test fmt vet deps help

# Build variables
BINARY_NAME=rtp-monitor
BUILD_DIR=./bin
VERSION?=0.1.0
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

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

# Format code
fmt:
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

# Create release builds for multiple platforms
release:
	@echo "Creating release builds..."
	@mkdir -p $(BUILD_DIR)/release
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 .
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Release builds created in $(BUILD_DIR)/release/"

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
	@echo "  fmt          - Format code"
	@echo "  vet          - Vet code"
	@echo "  lint         - Lint code (requires golangci-lint)"
	@echo "  clean        - Clean build artifacts"
	@echo "  dev-build    - Build development version with race detector"
	@echo "  release      - Create release builds for multiple platforms"
	@echo "  security     - Check for security vulnerabilities"
	@echo "  docs         - Generate documentation"
	@echo "  help         - Show this help message"
