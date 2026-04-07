# Configuration Reference

## Sources and precedence

Runtime configuration is resolved in this order (highest first):

1. CLI flags
2. Config file (`--config` YAML/JSON)
3. Built-in defaults

Write policy has an additional layer:

- Environment variables are loaded by default.
- Config `write_policy` values override corresponding environment fields.

## Core CLI flags

- `--config`: path to YAML/JSON config
- `--tag-map-csv`: path to tag CSV mapping
- `--transport`: `stdio|sse|streamable`
- `--modbus-ip`, `--modbus-port`
- `--modbus-timeout`, `--modbus-idle-timeout`
- `--modbus-retry-attempts`, `--modbus-retry-backoff`, `--modbus-retry-on-write`
- `--modbus-circuit-trip-after`, `--modbus-circuit-open-for`
- `--mock-mode`, `--mock-registers`, `--mock-coils`

## Config file keys

```yaml
modbus_ip: 192.168.1.22
modbus_port: 5002
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

## Write policy environment variables

- `MODBUS_WRITES_ENABLED`
- `MODBUS_WRITE_ALLOWLIST`
- `MODBUS_WRITE_ALLOWLIST_HOLDING`
- `MODBUS_WRITE_ALLOWLIST_COILS`
