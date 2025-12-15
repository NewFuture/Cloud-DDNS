.PHONY: build run clean docker docker-up docker-down test help

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
	@go clean
	@echo "Clean complete!"

# Build Docker image
docker:
	@echo "Building Docker image..."
	@docker build -t cloud-ddns:latest .
	@echo "Docker image built!"

# Start with Docker Compose
docker-up:
	@echo "Starting with Docker Compose..."
	@docker-compose up -d
	@echo "Service started!"

# Stop Docker Compose
docker-down:
	@echo "Stopping Docker Compose..."
	@docker-compose down
	@echo "Service stopped!"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete!"

# Run go mod tidy
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "Tidy complete!"

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  run         - Build and run the application"
	@echo "  clean       - Remove build artifacts"
	@echo "  docker      - Build Docker image"
	@echo "  docker-up   - Start with Docker Compose"
	@echo "  docker-down - Stop Docker Compose"
	@echo "  fmt         - Format Go code"
	@echo "  tidy        - Tidy Go modules"
	@echo "  help        - Show this help message"
