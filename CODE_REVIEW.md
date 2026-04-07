# Code Review Notes

This document records architecture and quality review findings and the remediation status.

## Addressed

### Architecture / SRP / DRY

- Tool handlers split by domain:
  - `modbus/tool_registers.go`
  - `modbus/tool_tags.go`
  - `modbus/tool_status.go`
  - `modbus/tool_helpers.go`
  - `modbus/tool_types.go`
- Configuration override flow refactored around `RuntimeOptions` to reduce long argument lists.

### Reliability

- Retry matching hardened:
  - case-insensitive transient string checks
  - wrapped timeout detection with `errors.As`
- Retry backoff now releases the client mutex while sleeping to avoid unnecessary contention.

### Safety / Correctness

- `write-tag` now rejects ambiguous inputs (e.g. both raw and typed values in one call).
- Tag scale semantics improved:
  - explicit `scale=0` is preserved
  - default scale only applies when scale is omitted

### Documentation

- Build commands corrected to `go build -o modbus-server .`
- Added references:
  - `CONFIG_REFERENCE.md`
  - `TAG_CSV_REFERENCE.md`
  - `ARCHITECTURE.md`
  - `SECURITY.md`

## Remaining Improvements (Future)

- Introduce transport-level integration tests (sse/streamable) in CI.
- Add proxy-capture correlation/export once proxy maturation branch is merged.
- Add typed write round-trip integration tests against a real Modbus target.
