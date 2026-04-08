# Go Modbus MCP Server

A lightweight, high-performance [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server for communicating with Modbus TCP devices. It exposes physical PLC hardware logic, registers, and coils to Large Language Models natively via MCP tools.

## Key Features

- **Official SDK**: Built securely on top of the official `github.com/modelcontextprotocol/go-sdk`.
- **Multiple Transports**: Fully supports `stdio`, Server-Sent Events (`sse`), and the new Streamable HTTP (`streamable`) MCP protocols.
- **Pluggable Drivers**: Supports `goburrow` and `simonvetter` drivers via `--modbus-driver`.
- **RTU + TCP Modes**: Supports both `tcp` and `rtu` bus modes via `--modbus-mode`.
- **Session-Safe Streamable HTTP**: Streamable transport runs in stateless JSON mode to avoid stale session issues (`session not found`) across reconnects.
- **Resilient Connections**: Thread-safe Modbus dialer reconnects safely for each operation and recovers cleanly from idle timeout disconnects.
- **Write Safety Guard**: Write tools are disabled by default and require explicit enablement via environment policy.
- **Auto-Recovery Controls**: Configurable retry/backoff and circuit protection for transient network failures.
- **Strongly Typed**: Input parsing schemas are automatically generated leveraging JSON Schema struct extraction.
- **Container Ready**: Includes health checks (`/health` on port 8080) and a multi-stage Docker build process.

## Available Tools

The server exposes the following MCP Tools to your LLM:

1. `read-holding-registers`: Read an array of `uint16` values from a specified address.
2. `read-coils`: Read an array of `boolean` flags from a specified digital input/output address.
3. `read-input-registers`: Read Modbus input registers.
4. `read-discrete-inputs`: Read Modbus discrete inputs.
5. `read-holding-registers-typed`: Read and decode typed holding-register values (`uint16`, `int16`, `uint32`, `int32`, `float32`, `string`) with optional byte/word order and scale/offset.
6. `write-holding-registers`: Write an array of `uint16` values sequentially starting at a specified address.
7. `write-holding-registers-typed`: Encode and write typed holding-register values (`uint16`, `int16`, `uint32`, `int32`, `float32`, `string`) with optional byte/word order and scale/offset.
8. `write-coils`: Write an array of `boolean` flags sequentially starting at a specified address.
9. `get-modbus-client-status`: Inspect retry counters, failures, and circuit state.
10. `list-tags`: List configured semantic tag definitions.
11. `read-tag`: Read by semantic tag name.
12. `write-tag`: Write by semantic tag name (subject to write policy), with raw arrays or typed values (`numeric_value`, `string_value`, `bool_value`).

## Quick Start

### Build & Run Locally

Ensure you have Go 1.25+ installed.

Toolchain policy: this repository is pinned to Go 1.25 across local development, CI, and Docker builds.

```bash
go build -o modbus-server .

# Connect to a default simulator (192.168.1.22:5002) using Streamable HTTP on port 8080
./modbus-server

# Load settings from YAML/JSON config file
./modbus-server --config ./server-config.yaml

# Or pass tag mapping CSV directly
./modbus-server --tag-map-csv ./tag-map.csv

# Run fully in-memory (no PLC required)
./modbus-server --mock-mode --config ./server-config.yaml

# Connect to a specific PLC IP and Port using STDIO (Best for Claude Desktop / Cursor)
./modbus-server --modbus-ip 10.0.0.50 --modbus-port 502 --transport stdio

# Use simonvetter driver over TCP
./modbus-server --modbus-driver simonvetter --modbus-mode tcp --modbus-ip 10.0.0.50 --modbus-port 502

# Use simonvetter driver over RTU serial
./modbus-server --modbus-driver simonvetter --modbus-mode rtu --serial-port /dev/ttyUSB0 --baud-rate 9600 --data-bits 8 --parity N --stop-bits 1

# Run using standard Server-Sent Events (SSE)
./modbus-server --transport sse

# Enable writes explicitly (disabled by default)
MODBUS_WRITES_ENABLED=true ./modbus-server

# Tune retries and circuit behavior
./modbus-server --modbus-retry-attempts 3 --modbus-retry-backoff 150ms --modbus-circuit-trip-after 3 --modbus-circuit-open-for 2s
```

Write policy environment variables:

- `MODBUS_WRITES_ENABLED`: `true|false` (default `false`)
- `MODBUS_WRITE_ALLOWLIST`: optional global address allowlist, e.g. `0-50,100,120-130`
- `MODBUS_WRITE_ALLOWLIST_HOLDING`: optional allowlist override for holding-register writes
- `MODBUS_WRITE_ALLOWLIST_COILS`: optional allowlist override for coil writes

You can also set write policy, retry settings, and tag-map CSV in config file (YAML/JSON). CLI flags override config file values.

Recommended: keep semantic tag definitions in CSV (`tag_map_csv`) rather than embedding tags directly in config.

Example `server-config.yaml`:

```yaml
modbus_ip: 192.168.1.22
modbus_port: 5002
modbus_driver: goburrow
modbus_mode: tcp
serial_port: /dev/ttyUSB0
baud_rate: 9600
data_bits: 8
parity: N
stop_bits: 1
transport: streamable
tag_map_csv: ./tag-map.csv
modbus_timeout: 10s
modbus_idle_timeout: 2s
modbus_retry_attempts: 3
modbus_retry_backoff: 150ms
modbus_retry_on_write: false
modbus_circuit_trip_after: 3
modbus_circuit_open_for: 2s
mock_mode: false
mock_registers: 1024
mock_coils: 1024

write_policy:
  writes_enabled: false
  write_allowlist: "0-9"
  holding_write_allowlist: "0-20"
  coil_write_allowlist: "0-5"
```

Tag CSV columns:

- Required: `name`, `kind`, `address`
- Optional: `quantity`, `slave_id`, `access`, `data_type`, `byte_order`, `word_order`, `scale`, `offset`, `description`

Supported `data_type` for holding-register tags:

- `uint16`, `int16`, `uint32`, `int32`, `float32`, `string`

Supported `data_type` for coil tags:

- `bool`

Typed `write-tag` inputs:

- `numeric_value` for numeric holding tags
- `string_value` for string holding tags
- `bool_value` for single-coil tags (`quantity=1`)

### Docker Usage

```bash
# Build the image
docker build -t go-mdbus-mcp .

# Run the server (exposes port 8080)
docker run -p 8080:8080 go-mdbus-mcp ./modbus-server --modbus-ip 10.0.0.50 --modbus-port 502 --transport sse
```

## Integrating with Claude Desktop

To use this server natively in Claude Desktop, add the following to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "modbus": {
      "command": "/path/to/your/compiled/modbus-server",
      "args": [
        "--transport", "stdio",
        "--modbus-ip", "192.168.1.22",
        "--modbus-port", "502"
      ]
    }
  }
}
```

## Contributing & Testing

Detailed usage and OpenCode validation steps are documented in `HOW_TO_USE.md`.

Historical validation run notes are archived in `docs/archive/TEST_DOCUMENT.md`.

Security deployment guidance is documented in `SECURITY.md`.

Configuration and precedence reference is documented in `CONFIG_REFERENCE.md`.

Tag CSV schema and typing reference is documented in `TAG_CSV_REFERENCE.md`.

Architecture overview is documented in `ARCHITECTURE.md`.

Detailed review notes and remediation status are documented in `CODE_REVIEW.md`.

Standard Go toolchains are used. The heavy lifting is done via the `modbus` folder.

```bash
go test ./...
make bench
```
