# How To Use And Validate With OpenCode

This document shows how to run the Modbus MCP server and validate it end-to-end using OpenCode tool calls.

## Prerequisites

- Modbus TCP target reachable (example: `192.168.1.22:5002`)
- `modbus-server` binary built from current source
- OpenCode session with access to this MCP server and Modbus tools

Build locally:

```bash
go build -o modbus-server main.go
```

Run server:

```bash
./modbus-server --transport streamable --modbus-ip 192.168.1.22 --modbus-port 5002
```

Or load from config file:

```bash
./modbus-server --config ./server-config.yaml
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

## What Was Verified In This Project

The OpenCode test session verified:

- Single register writes now use FC06 and read back correctly.
- Multi-register writes/reads work end-to-end with FC10/FC03.
- Coil write/read works for single and multiple values.
- Validation errors are returned for `quantity=0` and empty write arrays.
- Reconnect behavior is stable with short Modbus idle timeouts.

## Useful Logs To Check

Server log indicators of healthy behavior:

- `modbus: sending ... 01 06 ...` for single register write
- `modbus: sending ... 01 10 ...` for multi-register write
- `modbus: closing connection due to idle timeout: ...`
- Subsequent requests succeed after reconnect
