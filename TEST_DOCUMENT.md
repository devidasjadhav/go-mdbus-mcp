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
