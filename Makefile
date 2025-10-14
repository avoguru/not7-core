.PHONY: build build-all test lint clean run-example

# Binary name
BINARY=not7
VERSION=0.1.0

# Build flags
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

# Build for current platform
build:
	@echo "Building NOT7..."
	go build $(LDFLAGS) -o $(BINARY) .
	@echo "✅ Build complete: ./$(BINARY)"

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	
	@echo "  - macOS (Intel)"
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 .
	
	@echo "  - macOS (ARM)"
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 .
	
	@echo "  - Linux (AMD64)"
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 .
	
	@echo "  - Linux (ARM64)"
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 .
	
	@echo "  - Windows (AMD64)"
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe .
	
	@echo "✅ All builds complete in ./dist/"

# Run tests
test:
	go test -v ./...

# Run linter
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: brew install golangci-lint" && exit 1)
	golangci-lint run

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/
	@echo "✅ Clean complete"

# Run example
run-example: build
	@if [ -z "$(OPENAI_API_KEY)" ]; then \
		echo "❌ OPENAI_API_KEY not set"; \
		exit 1; \
	fi
	./$(BINARY) run examples/poem-generator.json

# Install dependencies (if needed)
deps:
	go mod download
	go mod tidy

# Display help
help:
	@echo "NOT7 Makefile Commands:"
	@echo "  make build        - Build for current platform"
	@echo "  make build-all    - Build for all platforms"
	@echo "  make test         - Run tests"
	@echo "  make lint         - Run linter"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make run-example  - Build and run example"
	@echo "  make deps         - Install dependencies"

