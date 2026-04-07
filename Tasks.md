# Migration Tasks: Modular Drivers + RTU + MCP Runtime Separation

This plan is grouped by implementation ease and executed in phases.
Each phase should end with a dedicated commit.

## Phase 1 (Easy): Foundational Refactor, No Behavior Change

Goal: introduce seams/interfaces while preserving current behavior.

- [x] Create internal `modbus` driver interface (read/write/status/close contract).
- [x] Wrap existing Goburrow implementation behind adapter (`goburrow` adapter).
- [x] Update tool handlers to depend on the driver interface instead of concrete client type.
- [x] Introduce `internal/logx` logger setup helper and route startup logger initialization through it.
- [x] Keep runtime behavior unchanged and verify with `go test ./...`.

Commit target:

- `refactor: add modbus driver interface and goburrow adapter`

## Phase 2 (Easy-Medium): Config and Runtime Modularization

Goal: reduce coupling in `main.go` and make transport/runtime wiring replaceable.

- [x] Move config loading/validation into `internal/config` package.
- [x] Add config schema fields for driver and bus mode:
  - `modbus_driver` (`goburrow|simonvetter`)
  - `modbus_mode` (`tcp|rtu`)
- [x] Add RTU serial config fields (port, baud rate, parity, data bits, stop bits).
- [x] Add thin MCP runtime wrapper package (`internal/mcpserver`) to isolate SDK-specific transport wiring.
- [x] Keep CLI flag precedence and existing behavior.

Commit target:

- `refactor: modularize config and mcp runtime wiring`

## Phase 3 (Medium): Add Simonvetter Driver (TCP First)

Goal: support alternate backend with parity on current TCP features.

- [x] Implement `simonvetter` TCP adapter that satisfies the driver interface.
- [x] Add `--modbus-driver` flag and config support.
- [x] Normalize adapter errors to internal error categories (timeout/connection/protocol/other).
- [x] Add parity tests to ensure equivalent behavior for key tools on both drivers.

Commit target:

- `feat: add simonvetter tcp driver adapter`

## Phase 4 (Medium-Hard): RTU Support

Goal: support serial Modbus RTU end-to-end through same tools.

- [ ] Implement RTU path in driver config/build flow.
- [ ] Add serial parameter validation and safe defaults.
- [ ] Ensure RTU lifecycle strategy (persistent serial with reconnect on failure).
- [ ] Add guarded integration tests for RTU (enabled only when test serial env vars are present).

Commit target:

- `feat: add modbus rtu mode via pluggable drivers`

## Phase 5 (Medium): Quality and Observability Hardening

Goal: improve operability and maintainability before default switch.

- [ ] Add structured operation logs (driver, mode, slave_id, function, latency, retries).
- [ ] Add expanded status reporting for active driver/mode and recent error class.
- [ ] Add soak-style reliability test harness for repeated operations.
- [ ] Update docs (`README.md`) with driver selection and RTU usage examples.

Commit target:

- `chore: improve observability and docs for driver/rtu support`

## Phase 6 (Hard): Default Driver Decision and Cleanup

Goal: finalize migration based on test outcomes.

- [ ] Run parity matrix and compare reliability/perf between drivers.
- [ ] Decide default driver (`goburrow` or `simonvetter`).
- [ ] Deprecate/remove temporary compatibility code if no longer needed.
- [ ] Publish migration notes and compatibility matrix.

Commit target:

- `chore: finalize driver migration defaults and cleanup`

---

## Execution Rules

- Complete phases in order.
- Run `go test ./...` at the end of every phase.
- Commit once per phase; avoid mixing unrelated changes.
- Keep `opencode.json` and `opencode.jsonc` local-only (do not commit).
