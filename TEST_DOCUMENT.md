# Test Document

This document logs validation runs executed against `go-mdbus-mcp`.

## Phase 1: Write Guarding & Safety Policy

Date: 2026-04-07

Scope validated:

- Writes disabled by default.
- Reads unaffected by write policy.
- Writes enabled explicitly by environment variable.
- Global allowlist enforcement.
- Range boundary enforcement for multi-value writes.
- Per-type allowlist override precedence.
- Invalid policy config fails at startup.

### Test Log

1) Default deny: write blocked

- Server startup mode: default (no `MODBUS_WRITES_ENABLED` set)
- Call: `write-holding-registers(address=0, values=[1], slave_id=1)`
- Expected: guarded rejection
- Actual: `guarded rejection: write blocked by policy (MODBUS_WRITES_ENABLED=false)`
- Result: PASS

2) Read path still works when writes blocked

- Call: `read-holding-registers(address=0, quantity=1, slave_id=1)`
- Expected: successful read
- Actual: `Holding registers at address 0: [1]`
- Result: PASS

3) Writes enabled: holding + coil write/readback

- Server startup mode:
  - `MODBUS_WRITES_ENABLED=true`
- Calls:
  - `write-holding-registers(address=0, values=[2], slave_id=1)`
  - `read-holding-registers(address=0, quantity=1, slave_id=1)`
  - `write-coils(address=0, values=[true], slave_id=1)`
  - `read-coils(address=0, quantity=1, slave_id=1)`
- Expected: writes and readback succeed
- Actual:
  - `Successfully wrote holding register at address 0: 2`
  - `Holding registers at address 0: [2]`
  - `Successfully wrote 1 values to coils starting at address 0: [true]`
  - `Coils at address 0: [true]`
- Result: PASS

4) Global allowlist enforcement

- Server startup mode:
  - `MODBUS_WRITES_ENABLED=true`
  - `MODBUS_WRITE_ALLOWLIST=0-9`
- Calls:
  - in-range: `write-holding-registers(address=5, values=[77], slave_id=1)`
  - out-of-range: `write-holding-registers(address=10, values=[88], slave_id=1)`
  - out-of-range coil: `write-coils(address=10, values=[true], slave_id=1)`
  - readback: `read-holding-registers(address=5, quantity=1, slave_id=1)`
- Expected:
  - address 5 write allowed
  - address 10 writes blocked
- Actual:
  - `Successfully wrote holding register at address 5: 77`
  - `guarded rejection: holding-register write range 10-10 not in allowlist`
  - `guarded rejection: coil write range 10-10 not in allowlist`
  - `Holding registers at address 5: [77]`
- Result: PASS

5) Multi-value boundary enforcement

- Server startup mode:
  - `MODBUS_WRITES_ENABLED=true`
  - `MODBUS_WRITE_ALLOWLIST=0-9`
- Call: `write-holding-registers(address=9, values=[11,12], slave_id=1)`
- Expected: blocked because range becomes `9-10`
- Actual: `guarded rejection: holding-register write range 9-10 not in allowlist`
- Result: PASS

6) Per-type override precedence

- Server startup mode:
  - `MODBUS_WRITES_ENABLED=true`
  - `MODBUS_WRITE_ALLOWLIST=0-9`
  - `MODBUS_WRITE_ALLOWLIST_HOLDING=0-20`
  - `MODBUS_WRITE_ALLOWLIST_COILS=0-5`
- Calls:
  - `write-holding-registers(address=15, values=[123], slave_id=1)`
  - `write-coils(address=15, values=[true], slave_id=1)`
  - `read-holding-registers(address=15, quantity=1, slave_id=1)`
- Expected:
  - holding write at 15 allowed by holding override
  - coil write at 15 blocked by coil override
- Actual:
  - `Successfully wrote holding register at address 15: 123`
  - `guarded rejection: coil write range 15-15 not in allowlist`
  - `Holding registers at address 15: [123]`
- Result: PASS

7) Invalid allowlist config fails fast

- Startup command A:
  - `MODBUS_WRITES_ENABLED=true MODBUS_WRITE_ALLOWLIST=a-b ./modbus-server --transport stdio`
- Expected: startup failure with parse error
- Actual:
  - `Invalid write policy configuration: invalid MODBUS_WRITE_ALLOWLIST: invalid range start "a-b"`
- Result: PASS

- Startup command B:
  - `MODBUS_WRITE_ALLOWLIST=1-2-3 ./modbus-server --transport stdio`
- Expected: startup failure with invalid token format
- Actual:
  - `Invalid write policy configuration: invalid MODBUS_WRITE_ALLOWLIST: invalid allowlist token "1-2-3"`
- Result: PASS

## Conclusion

Phase 1 write guarding behavior is validated and working as designed across normal, boundary, override, and invalid-configuration scenarios.

## Phase 2: Connection Recovery + Config-Driven Runtime

Date: 2026-04-07

Scope validated:

- Config file loading from `--config` (YAML/JSON parser path)
- CLI flag precedence over config file values
- Fail-fast behavior for invalid duration fields
- Retry classifier behavior for transient errors
- Write policy parser/guard behavior via unit tests

### Test Log

1) Build + startup smoke with config

- Command:
  - `go build -o modbus-server .`
  - `timeout 3 ./modbus-server --config ./server-config.yaml --transport stdio`
- Expected:
  - server starts successfully
  - write policy banner printed
- Actual:
  - startup succeeded
  - banner printed: writes disabled by default
- Result: PASS

2) Invalid config duration fails fast

- Command:
  - `modbus_retry_backoff: nope` in temp config
  - `./modbus-server --config /tmp/bad-config.yaml --transport stdio`
- Expected:
  - startup error with clear duration parse failure
- Actual:
  - `Invalid config value: invalid modbus_retry_backoff "nope": time: invalid duration "nope"`
- Result: PASS

3) Automated unit tests for policy/retry/config

- Added tests:
  - `modbus/write_policy_test.go`
  - `modbus/client_retry_test.go`
  - `config_test.go`
- Command:
  - `go test ./...`
- Expected:
  - all tests pass
- Actual:
  - `ok github.com/devidasjadhav/go-mdbus-mcp`
  - `ok github.com/devidasjadhav/go-mdbus-mcp/modbus`
- Result: PASS

## Conclusion (Phase 1 + Phase 2)

The server is validated for safety gating and config-driven runtime controls. Core policy and configuration paths are covered with both runtime checks and unit tests.

## Phase 3: CSV Tag Mapping + Typed Tag Reads

Date: 2026-04-07

Scope validated:

- CSV-based tag loading (`--tag-map-csv` and `tag_map_csv` in config)
- Data type decoding for `float32` and `string` on holding-register tags
- Required CSV header validation and fail-fast behavior
- Quantity derivation from data type (`float32` => quantity `2` when omitted)
- Makefile run-argument behavior with config compatibility

### Test Log

1) Unit tests for CSV mapping and codec behavior

- Added/updated tests:
  - `modbus/tag_codec_test.go`
  - `config_test.go` (CSV load, quantity derivation, missing required column)
- Command:
  - `go test ./...`
- Actual:
  - `ok github.com/devidasjadhav/go-mdbus-mcp`
  - `ok github.com/devidasjadhav/go-mdbus-mcp/modbus`
- Result: PASS

2) Runtime smoke with config + CSV mapping

- Commands:
  - `go build -o modbus-server .`
  - `timeout 4 ./modbus-server --config ./server-config.yaml --transport stdio`
- Expected:
  - tag map loaded and startup banner shows count
- Actual:
  - `Loaded 4 configured tags`
- Result: PASS

3) Makefile run target behavior with config

- Command:
  - `make run CONFIG=./server-config.yaml TRANSPORT=stdio ARGS="--version"`
- Expected:
  - should not force default IP/port flags that override config
- Actual:
  - command uses `--transport` and `--config` only (plus extra args)
- Result: PASS

## Conclusion (Phase 1 + Phase 2 + Phase 3)

Safety gating, recovery controls, and CSV-driven semantic tags with typed reads are validated through unit tests and runtime smoke checks.

## Phase 4: Typed Tag Writes

Date: 2026-04-07

Scope validated:

- `write-tag` supports typed inputs in addition to raw arrays
  - `numeric_value` for numeric holding tags
  - `string_value` for string holding tags
  - `bool_value` for single-coil tags
- Encoding paths for `float32` and `string` are covered by unit tests

### Test Log

1) Unit tests for typed encode/decode paths

- Added/updated tests:
  - `modbus/tag_codec_test.go`
- Covered:
  - decode `float32`
  - decode `string`
  - encode `float32`
  - encode `string`
- Command:
  - `go test ./...`
- Actual:
  - `ok github.com/devidasjadhav/go-mdbus-mcp`
  - `ok github.com/devidasjadhav/go-mdbus-mcp/modbus`
- Result: PASS

## Conclusion (Phase 1-4)

The server now supports guarded writes, recovery controls, CSV-based semantic tags, typed tag reads, and typed tag writes with automated test coverage.

## Phase 5: Mock Mode (Hardware-Free Validation)

Date: 2026-04-07

Scope validated:

- In-memory Modbus client enabled via `--mock-mode`
- Mock register and coil read/write behavior
- Config support for `mock_mode`, `mock_registers`, `mock_coils`

### Test Log

1) Unit tests for in-memory Modbus behavior

- Added tests:
  - `modbus/mock_client_test.go`
- Covered:
  - holding register write/read
  - coil write/read (packed bit format)
- Command:
  - `go test ./...`
- Result: PASS

## Conclusion (Phase 1-5)

The server now includes a deterministic in-memory mock path for development and CI, reducing dependence on physical hardware for core tool validation.

## Phase 6: Benchmark Harness (Mock Path)

Date: 2026-04-07

Scope validated:

- Added benchmark suite in `modbus/benchmark_test.go`
- Added `make bench` target for repeatable benchmark runs
- Captured baseline numbers for mock-mode execution path

### Test Log

1) Benchmark execution

- Command:
  - `go test -bench=. -benchmem ./modbus`
- Actual baseline on this environment:
  - `BenchmarkMockRead4Registers-12`: `29.65 ns/op`, `8 B/op`, `1 allocs/op`
  - `BenchmarkMockWriteAndReadVerify-12`: `42.63 ns/op`, `8 B/op`, `2 allocs/op`
- Result: PASS

Note:

- These are mock-path microbenchmarks (in-memory), useful for regression tracking of server overhead.
- Hardware/network latencies are expected to be significantly higher in real Modbus TCP environments.

## Conclusion (Phase 1-6)

The project now includes safety controls, recovery logic, CSV semantic tags with typed reads/writes, hardware-free mock mode, and a benchmark baseline for regression tracking.
