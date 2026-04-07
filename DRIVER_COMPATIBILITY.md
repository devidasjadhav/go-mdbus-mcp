# Driver Compatibility Matrix

Date: 2026-04-07

This document captures the current driver decision and compatibility status.

## Default Driver Decision

- Default driver remains `goburrow` for TCP workloads.
- `simonvetter` is supported and recommended when RTU serial mode is required.

Reasoning:

- `goburrow` path has been in use longer in this project and remains the safest default for existing users.
- `simonvetter` adds broad transport capabilities and RTU support behind the same driver interface.
- Keeping `goburrow` as default avoids surprise regressions during migration.

## Matrix

| Capability | goburrow | simonvetter |
|---|---|---|
| Driver selection (`--modbus-driver`) | Yes | Yes |
| TCP mode (`--modbus-mode tcp`) | Yes | Yes |
| RTU mode (`--modbus-mode rtu`) | Planned under adapter expansion | Yes |
| Holding registers read/write | Yes | Yes |
| Input registers read | Yes | Yes |
| Coils/discrete inputs read/write | Yes | Yes |
| Retry/backoff/circuit integration | Yes | Yes |
| Status reporting (driver/mode/error category) | Yes | Yes |
| Mock mode parity tests | Yes | Yes (through shared mock path) |

## Validation Coverage

- Unit tests:
  - driver selection, defaults, and validation paths
  - retry, codec, write policy, tag map
- Integration tests:
  - stdio + streamable MCP calls in mock mode
  - parity path with `--modbus-driver simonvetter` in mock mode
- RTU integration:
  - guarded hardware test (`MODBUS_RTU_TEST_PORT` env var)

## Migration Guidance

- Existing users: no changes needed; default remains `goburrow`.
- RTU users: switch to `--modbus-driver simonvetter --modbus-mode rtu`.
- Recommended rollout:
  1. Validate in staging with `simonvetter`.
  2. Compare status metrics and error categories.
  3. Promote to production if stable.
