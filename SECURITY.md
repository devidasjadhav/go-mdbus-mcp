# Security Guidelines

This project can write to physical control points. Treat deployment and access as production-critical.

## Safe Defaults

- Writes are disabled by default (`MODBUS_WRITES_ENABLED=false`).
- Enable writes only when required.
- Prefer address allowlists when writes are enabled:
  - `MODBUS_WRITE_ALLOWLIST`
  - `MODBUS_WRITE_ALLOWLIST_HOLDING`
  - `MODBUS_WRITE_ALLOWLIST_COILS`

## Least Privilege

- Run as a non-root service account.
- Prefer non-privileged ports (for example `1502`) where possible.
- Restrict network ACLs so only trusted clients can reach MCP and Modbus endpoints.

## Production Recommendations

- Use `--config` with explicit policy and retry settings.
- Keep tag mappings in controlled CSV files with change review.
- Log and audit all write operations.
- Use mock mode in CI/staging to avoid accidental hardware writes.

## Incident Reduction Checklist

- Keep `MODBUS_WRITES_ENABLED=false` in read-only environments.
- Use explicit allowlists for any write-enabled environment.
- Verify `slave_id` and address ranges before rollout.
- Roll out policy changes in staging first with mock mode and tests.
