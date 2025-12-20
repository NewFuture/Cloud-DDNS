.PHONY: build run clean docker test test-verbose test-coverage test-race fmt fmt-check vet tidy help

# Binary name
BINARY_NAME=cloud-ddns

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) main.go
	@echo "Build complete!"

# Run the application
run: build
	@echo "Starting $(BINARY_NAME)..."
	@./$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.txt coverage.html
	@go clean
	@echo "Clean complete!"

# Run tests
test:
	@echo "Running tests..."
	@go test ./...
	@echo "Tests complete!"

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.txt -covermode=atomic ./...
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detector (separate from coverage due to platform limitations)
test-race:
	@echo "Running tests with race detector..."
	@go test -v -race ./...
	@echo "Race detection complete!"

# Build Docker image
docker:
	@echo "Building Docker image..."
	@docker build -t cloud-ddns:latest .
	@echo "Docker image built!"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete!"

# Check code formatting
fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not properly formatted:"; \
		gofmt -l .; \
		echo "Please run 'make fmt' to format the code"; \
		exit 1; \
	fi
	@echo "All files are properly formatted!"

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "Vet complete!"

# Run go mod tidy
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "Tidy complete!"

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  run            - Build and run the application"
	@echo "  clean          - Remove build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-race      - Run tests with race detector"
	@echo "  docker         - Build Docker image"
	@echo "  fmt            - Format Go code"
	@echo "  fmt-check      - Check code formatting without modifying files"
	@echo "  vet            - Run go vet"
	@echo "  tidy           - Tidy Go modules"
	@echo "  help           - Show this help message"
