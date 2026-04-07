# Architecture Overview

## Runtime flow

1. `main.go` parses flags, loads optional config, and applies precedence.
2. Write policy is loaded and validated.
3. Tag map is loaded from CSV or inline config.
4. Modbus client is created (TCP or in-memory mock).
5. MCP server is initialized and tools are registered.
6. Transport (`stdio`, `sse`, or `streamable`) is started.

## Core modules

- `main.go`: bootstrap, transport setup, lifecycle
- `config.go`: config parsing and CSV tag loading
- `modbus/client.go`: Modbus client execution, retry/circuit status
- `modbus/write_policy.go`: write-guarding and allowlists
- `modbus/tag_map.go`: semantic tag model and validation
- `modbus/tag_codec.go`: typed decode/encode helpers
- `modbus/tools.go`: MCP tool handlers
- `modbus/mock_client.go`: deterministic in-memory Modbus implementation

## Design principles

- Safe defaults (writes disabled)
- Explicit policy for mutating operations
- Deterministic mock path for CI and local development
- CSV-first semantic mapping for maintainability
