.PHONY: help build test clean docker-build

# Default target
help:
	@echo "Available commands:"
	@echo "  build         Build the binary"
	@echo "  test          Run tests"
	@echo "  clean         Clean build artifacts"
	@echo "  docker-build  Build Docker image"

build:
	go build -ldflags="-s -w -X main.version=dev" -o modbus-server main.go

test:
	go test -v ./...

clean:
	rm -f modbus-server
	rm -f modbus-server-*
	rm -f coverage.out coverage.html

docker-build:
	docker build -t modbus-mcp-server .
