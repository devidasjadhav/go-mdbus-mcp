# Makefile for Modbus MCP Server

.PHONY: help build test clean docker-build docker-run release

# Default target
help: ## Show this help message
	@echo "Modbus MCP Server - Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

# Build targets
build: ## Build the binary for current platform
	@echo "🔨 Building Modbus MCP Server..."
	go build -ldflags="-s -w -X main.version=dev" -o modbus-server main.go
	@echo "✅ Build complete: modbus-server"

build-all: ## Build for multiple platforms
	@echo "🔨 Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o modbus-server-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o modbus-server-linux-arm64 main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o modbus-server-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o modbus-server-darwin-arm64 main.go
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o modbus-server-windows-amd64.exe main.go
	@echo "✅ Multi-platform build complete"

# Test targets
test: ## Run all tests
	@echo "🧪 Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "🧪 Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

# Development targets
fmt: ## Format Go code
	@echo "📝 Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	go vet ./...

lint: ## Run golangci-lint (if installed)
	@echo "🔍 Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

mod-tidy: ## Clean up go.mod and go.sum
	@echo "🧹 Tidying modules..."
	go mod tidy

# Docker targets
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t modbus-mcp-server .
	@echo "✅ Docker image built: modbus-mcp-server"

docker-run: ## Run Docker container
	@echo "🐳 Running Docker container..."
	docker run -p 8080:8080 --rm modbus-mcp-server

docker-test: ## Test the Docker container
	@echo "🧪 Testing Docker container..."
	docker run --rm modbus-mcp-server --version
	docker run --rm modbus-mcp-server --help

# Run targets
run: ## Run the server with default settings
	@echo "🚀 Starting Modbus MCP Server..."
	./modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002

run-dev: ## Run in development mode
	@echo "🚀 Starting in development mode..."
	go run main.go --modbus-ip 127.0.0.1 --modbus-port 502

# Integration test
test-integration: ## Run integration tests with Gemini
	@echo "🤖 Running Gemini integration tests..."
	@if [ ! -f "test_gemini.py" ]; then \
		echo "❌ test_gemini.py not found"; \
		exit 1; \
	fi
	python test_gemini.py

# Clean targets
clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	rm -f modbus-server
	rm -f modbus-server-*
	rm -f *.exe
	rm -f coverage.out coverage.html

clean-all: clean ## Clean all artifacts including Docker
	@echo "🧹 Cleaning all artifacts..."
	docker rmi modbus-mcp-server 2>/dev/null || true

# Release targets
release-patch: ## Create a patch release (0.0.x)
	@echo "📦 Creating patch release..."
	git tag -a $$(git describe --tags --abbrev=0 | awk -F. '{print $$1"."$$2"."$$3+1}') -m "Release $$(git describe --tags --abbrev=0 | awk -F. '{print $$1"."$$2"."$$3+1}')"
	git push origin --tags

release-minor: ## Create a minor release (0.x.0)
	@echo "📦 Creating minor release..."
	git tag -a $$(git describe --tags --abbrev=0 | awk -F. '{print $$1"."$$2+1".0"}') -m "Release $$(git describe --tags --abbrev=0 | awk -F. '{print $$1"."$$2+1".0"}')"
	git push origin --tags

release-major: ## Create a major release (x.0.0)
	@echo "📦 Creating major release..."
	git tag -a $$(git describe --tags --abbrev=0 | awk -F. '{print $$1+1".0.0"}') -m "Release $$(git describe --tags --abbrev=0 | awk -F. '{print $$1+1".0.0"}')"
	git push origin --tags

# Info targets
version: ## Show current version
	@echo "📋 Current version info:"
	@echo "  Git tag: $$(git describe --tags --abbrev=0 2>/dev/null || echo 'No tags')"
	@echo "  Git commit: $$(git rev-parse --short HEAD)"
	@echo "  Go version: $$(go version)"

info: ## Show project information
	@echo "📊 Project Information:"
	@echo "  Name: Modbus MCP Server"
	@echo "  Repository: https://github.com/devidasjadhav/go-mdbus-mcp"
	@echo "  Go Version: $$(go version | cut -d' ' -f3)"
	@echo "  Platform: $$(go env GOOS)/$(go env GOARCH)"
	@echo "  Module: $$(go list -m)"

# Development setup
setup: ## Setup development environment
	@echo "🔧 Setting up development environment..."
	go mod download
	@echo "✅ Dependencies downloaded"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "✅ golangci-lint already installed"; \
	else \
		echo "📦 Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi

# CI/CD simulation
ci: ## Run CI pipeline locally
	@echo "🔄 Running CI pipeline..."
	$(MAKE) mod-tidy
	$(MAKE) fmt
	$(MAKE) vet
	$(MAKE) test
	$(MAKE) build
	@echo "✅ CI pipeline completed successfully"