# How To Use And Validate With OpenCode

This document shows how to run the Modbus MCP server and validate it end-to-end using OpenCode tool calls.

## Prerequisites

- Modbus TCP target reachable (example: `192.168.1.22:5002`)
- `modbus-server` binary built from current source
- OpenCode session with access to this MCP server and Modbus tools

Build locally:

```bash
go build -o modbus-server .
```

Run server:

```bash
./modbus-server --transport streamable --modbus-ip 192.168.1.22 --modbus-port 5002
```

Or load from config file:

```bash
./modbus-server --config ./server-config.yaml
```

Or load semantic tags from CSV directly:

```bash
./modbus-server --tag-map-csv ./tag-map.csv
```

For hardware-free testing:

```bash
./modbus-server --mock-mode --config ./server-config.yaml
```

If you need write tools during validation, enable them explicitly:

```bash
MODBUS_WRITES_ENABLED=true ./modbus-server --transport streamable --modbus-ip 192.168.1.22 --modbus-port 5002
```

## OpenCode Validation Flow

Use these MCP tool calls from OpenCode.

### 1) Holding register single-value path

1. Read register 0:
   - `modbus_read-holding-registers(address=0, quantity=1, slave_id=1)`
2. Write register 0 to 1:
   - `modbus_write-holding-registers(address=0, values=[1], slave_id=1)`
3. Read register 0 again and verify `1`.

Expected protocol behavior:
- Single-register write uses function code `0x06` (`Write Single Register`).

### 2) Holding register multi-value path

1. Write 10 values starting at 0:
   - `modbus_write-holding-registers(address=0, values=[1,1,1,1,1,1,1,1,1,1], slave_id=1)`
2. Read 10 values starting at 0:
   - `modbus_read-holding-registers(address=0, quantity=10, slave_id=1)`

Expected protocol behavior:
- Multi-register write uses function code `0x10` (`Write Multiple Registers`).

### 3) Coil path

1. Write coil pattern:
   - `modbus_write-coils(address=0, values=[true,false,true,false,true,false,true,false,true,false], slave_id=1)`
2. Read 10 coils:
   - `modbus_read-coils(address=0, quantity=10, slave_id=1)`
3. Verify response matches pattern.

### 4) Input validation checks

Run these negative tests and verify errors are returned:

- `modbus_read-holding-registers(address=0, quantity=0, slave_id=1)`
  - Expected: `quantity must be greater than 0`
- `modbus_read-coils(address=0, quantity=0, slave_id=1)`
  - Expected: `quantity must be greater than 0`
- `modbus_write-holding-registers(address=0, values=[], slave_id=1)`
  - Expected: `values must contain at least one register value`
- `modbus_write-coils(address=0, values=[], slave_id=1)`
  - Expected: `values must contain at least one coil value`

### 5) Idle timeout resilience

1. Perform a successful read or write.
2. Wait longer than server idle timeout (example: 4 seconds).
3. Perform another read.

Expected result:
- Operation still succeeds.
- Server logs show clean close/reconnect between calls.

### 6) Client recovery status

Use this tool to inspect retry/circuit counters when diagnosing network instability:

- `get-modbus-client-status()`

### 7) Tag-based operations

With `tags` configured in `server-config.yaml`, use semantic tag tools:

- `list-tags()`
- `read-tag(name="ambient_temp_raw")`
- `write-tag(name="run_command", coil_values=[true])`

Typed write examples:

- `write-tag(name="boiler_temp_c", numeric_value=42.5)`
- `write-tag(name="device_label", string_value="PUMP-A")`
- `write-tag(name="run_command", bool_value=true)`

For typed reads, define `data_type` in CSV (`float32`, `string`, etc.).

Notes:

- `write-tag` is still guarded by write policy.
- `write-tag` value array length must match configured tag `quantity`.

### 8) Typed holding-register write + raw read-back

Use this when you want to write `float32`, `int32`, `uint32`, `int16`, `uint16`, or `string` directly without creating a tag.

Examples from OpenCode:

- `modbus_write-holding-registers-typed(address=100, data_type="float32", numeric_value=12.5)`
- `modbus_read-holding-registers(address=100, quantity=2)`
- `modbus_write-holding-registers-typed(address=110, data_type="string", quantity=2, string_value="ABC")`
- `modbus_read-holding-registers(address=110, quantity=2)`

Expected behavior:

- Typed write returns both logical input and encoded raw register values.
- Raw read shows the exact register words that were written.

### 9) Curl JSON-RPC validation (streamable transport)

Start server in mock mode with writes enabled:

```bash
MODBUS_WRITES_ENABLED=true ./modbus-server --mock-mode --transport streamable
```

Initialize MCP session:

```bash
curl -X POST "http://127.0.0.1:8080/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  --data '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"curl-test","version":"1"}},"id":1}'
```

Write a string and validate each register individually:

```bash
# Write "HELLO" across 3 registers
curl -X POST "http://127.0.0.1:8080/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  --data '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"write-holding-registers-typed","arguments":{"address":200,"data_type":"string","quantity":3,"string_value":"HELLO"}},"id":2}'

# Read each register separately
curl -X POST "http://127.0.0.1:8080/mcp" -H "Content-Type: application/json" -H "Accept: application/json, text/event-stream" --data '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read-holding-registers","arguments":{"address":200,"quantity":1}},"id":3}'
curl -X POST "http://127.0.0.1:8080/mcp" -H "Content-Type: application/json" -H "Accept: application/json, text/event-stream" --data '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read-holding-registers","arguments":{"address":201,"quantity":1}},"id":4}'
curl -X POST "http://127.0.0.1:8080/mcp" -H "Content-Type: application/json" -H "Accept: application/json, text/event-stream" --data '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read-holding-registers","arguments":{"address":202,"quantity":1}},"id":5}'
```

Expected individual words for `"HELLO"`:

- register `200`: `18501` (`'H' 'E'`)
- register `201`: `19532` (`'L' 'L'`)
- register `202`: `20224` (`'O' '\x00'`)

Optional typed read verification:

```bash
curl -X POST "http://127.0.0.1:8080/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  --data '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read-holding-registers-typed","arguments":{"address":200,"quantity":3,"data_type":"string"}},"id":6}'
```

## What Was Verified In This Project

The OpenCode test session verified:

- Single register writes now use FC06 and read back correctly.
- Multi-register writes/reads work end-to-end with FC10/FC03.
- Coil write/read works for single and multiple values.
- Typed holding-register writes (`write-holding-registers-typed`) work for numeric and string data types.
- Raw read-back (`read-holding-registers`) validates encoded register words, including individual register reads for string payloads.
- Validation errors are returned for `quantity=0` and empty write arrays.
- Reconnect behavior is stable with short Modbus idle timeouts.

## Useful Logs To Check

Server log indicators of healthy behavior:

- `modbus: sending ... 01 06 ...` for single register write
- `modbus: sending ... 01 10 ...` for multi-register write
- `modbus: closing connection due to idle timeout: ...`
- Subsequent requests succeed after reconnect
