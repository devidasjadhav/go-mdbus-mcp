.PHONY: help build test bench clean docker-build run run-streamable run-stdio run-sse stage1 stage2 stress stress-quick report

BINARY := modbus-server
MODBUS_IP ?=
MODBUS_PORT ?=
TRANSPORT ?= streamable
CONFIG ?=
ARGS ?=
ARTIFACT_DIR ?= /tmp/modbus-mcp-artifacts
REPORT_PATH ?= $(ARTIFACT_DIR)/comparison-report.md
STAGE3_DURATION ?= 3s
STAGE3_CONCURRENCY ?= 1,5,10

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
	@echo "  bench         Run benchmark suite (mock mode)"
	@echo "  clean         Clean build artifacts"
	@echo "  docker-build  Build Docker image"
	@echo "  run           Run server (configurable args)"
	@echo "  run-streamable Run server with streamable transport"
	@echo "  run-stdio     Run server with stdio transport"
	@echo "  run-sse       Run server with SSE transport"
	@echo "  stage1        Run Stage-1 matrix tests"
	@echo "  stage2        Run Stage-2 advanced matrix tests"
	@echo "  stress        Run Stage-3 stress matrix"
	@echo "  stress-quick  Run Stage-3 quick stress sanity"
	@echo "  report        Run Stage1+2+3 and generate markdown report"
	@echo ""
	@echo "Run variables (override with make VAR=value):"
	@echo "  MODBUS_IP     Optional (default from app/config)"
	@echo "  MODBUS_PORT   Optional (default from app/config)"
	@echo "  TRANSPORT     Default: $(TRANSPORT)"
	@echo "  CONFIG        Optional config file path"
	@echo "  ARGS          Extra raw CLI args"
	@echo "  ARTIFACT_DIR  Artifact output directory (default: $(ARTIFACT_DIR))"
	@echo "  REPORT_PATH   Markdown report output path (default: $(REPORT_PATH))"
	@echo "  STAGE3_DURATION   Stress duration (default: $(STAGE3_DURATION))"
	@echo "  STAGE3_CONCURRENCY Stress conc list (default: $(STAGE3_CONCURRENCY))"
	@echo ""
	@echo "Examples:"
	@echo "  make run"
	@echo "  make run CONFIG=./server-config.yaml"
	@echo "  make run MODBUS_IP=192.168.1.22 MODBUS_PORT=5002"
	@echo "  make run TRANSPORT=stdio"
	@echo "  make run ARGS=\"--tag-map-csv ./tag-map.csv\""
	@echo "  make report ARTIFACT_DIR=/tmp/modbus-report"

build:
	go build -ldflags="-s -w -X main.version=dev" -o $(BINARY) .

test:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./modbus

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

stage1:
	mkdir -p "$(ARTIFACT_DIR)"
	STAGE1_ARTIFACT_DIR="$(ARTIFACT_DIR)" go test -v . -run TestStage1MatrixMockMode

stage2:
	mkdir -p "$(ARTIFACT_DIR)"
	STAGE2_ARTIFACT_DIR="$(ARTIFACT_DIR)" go test -v . -run TestStage2AdvancedMatrixMockMode

stress:
	mkdir -p "$(ARTIFACT_DIR)"
	STAGE3_STRESS=1 STAGE3_ARTIFACT_DIR="$(ARTIFACT_DIR)" STAGE3_DURATION="$(STAGE3_DURATION)" STAGE3_CONCURRENCY="$(STAGE3_CONCURRENCY)" go test -v . -run TestStage3StressHarness

stress-quick:
	mkdir -p "$(ARTIFACT_DIR)"
	STAGE3_STRESS=1 STAGE3_QUICK=1 STAGE3_ARTIFACT_DIR="$(ARTIFACT_DIR)" go test -v . -run TestStage3StressHarness

report: stage1 stage2 stress
	go run ./tools/reportgen -server go-mdbus-mcp -stage1 "$(ARTIFACT_DIR)/stage1-results.json" -stage2 "$(ARTIFACT_DIR)/stage2-results.json" -stage3 "$(ARTIFACT_DIR)/stage3-results.json" -out "$(REPORT_PATH)"
	@echo "Report written: $(REPORT_PATH)"
