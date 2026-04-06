.PHONY: help build test clean docker-build run-streamable

# Default target
help:
	@echo "Available commands:"
	@echo "  build         Build the binary"
	@echo "  test          Run tests"
	@echo "  clean         Clean build artifacts"
	@echo "  docker-build  Build Docker image"
	@echo "  run-streamable  Run MCP server on :8080"

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

run-streamable:
	./modbus-server --transport streamable --modbus-ip 127.0.0.1 --modbus-port 5002
