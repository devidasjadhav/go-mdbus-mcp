.PHONY: help build test clean docker-build run run-streamable run-stdio run-sse

BINARY := modbus-server
MODBUS_IP ?=
MODBUS_PORT ?=
TRANSPORT ?= streamable
CONFIG ?=
ARGS ?=

RUN_ARGS := --transport $(TRANSPORT)
ifneq ($(strip $(MODBUS_IP)),)
RUN_ARGS += --modbus-ip $(MODBUS_IP)
endif
ifneq ($(strip $(MODBUS_PORT)),)
RUN_ARGS += --modbus-port $(MODBUS_PORT)
endif
ifneq ($(strip $(CONFIG)),)
RUN_ARGS += --config $(CONFIG)
endif
RUN_ARGS += $(ARGS)

# Default target
help:
	@echo "Available commands:"
	@echo "  build         Build the binary"
	@echo "  test          Run tests"
	@echo "  clean         Clean build artifacts"
	@echo "  docker-build  Build Docker image"
	@echo "  run           Run server (configurable args)"
	@echo "  run-streamable Run server with streamable transport"
	@echo "  run-stdio     Run server with stdio transport"
	@echo "  run-sse       Run server with SSE transport"
	@echo ""
	@echo "Run variables (override with make VAR=value):"
	@echo "  MODBUS_IP     Optional (default from app/config)"
	@echo "  MODBUS_PORT   Optional (default from app/config)"
	@echo "  TRANSPORT     Default: $(TRANSPORT)"
	@echo "  CONFIG        Optional config file path"
	@echo "  ARGS          Extra raw CLI args"
	@echo ""
	@echo "Examples:"
	@echo "  make run"
	@echo "  make run CONFIG=./server-config.yaml"
	@echo "  make run MODBUS_IP=192.168.1.22 MODBUS_PORT=5002"
	@echo "  make run TRANSPORT=stdio"
	@echo "  make run ARGS=\"--tag-map-csv ./tag-map.csv\""

build:
	go build -ldflags="-s -w -X main.version=dev" -o $(BINARY) .

test:
	go test -v ./...

clean:
	rm -f $(BINARY)
	rm -f modbus-server-*
	rm -f coverage.out coverage.html

docker-build:
	docker build -t modbus-mcp-server .

run: build
	./$(BINARY) $(RUN_ARGS)

run-streamable: TRANSPORT=streamable
run-streamable: run

run-stdio: TRANSPORT=stdio
run-stdio: run

run-sse: TRANSPORT=sse
run-sse: run
