# AGENTS.md

## Snapshot
- This is a single Go module (`go.mod`) with one executable entrypoint: `main.go`.
- Core package boundaries: `internal/config` (config load/merge/validation), `internal/mcpserver` (transport runtime), `modbus` (drivers + MCP tool handlers).
- Tool registration is centralized in `modbus/RegisterTools` (`modbus/tools.go`), then split into `tool_registers.go`, `tool_status.go`, `tool_tags.go`.

## Commands That Matter
- Build local binary: `go build -o modbus-server .`
- Run full tests (same as Makefile/CI): `go test -v ./...`
- Run focused package tests: `go test -v ./modbus -run <Regex>`
- Run root integration tests only: `go test -v . -run TestIntegration`
- Run benchmarks: `go test -bench=. -benchmem ./modbus` (or `make bench`)
- Convenience targets: `make build`, `make test`, `make run`, `make run-streamable`, `make run-stdio`, `make run-sse`

## Runtime + Config Gotchas
- Default transport is `streamable`; HTTP listens on `0.0.0.0:8080`, MCP endpoint is `/mcp`, health endpoint is `/health`.
- For `stdio` transport, logs intentionally go to stderr (`internal/logx`) to avoid corrupting JSON-RPC on stdout.
- Config precedence is strict: CLI flags override config file values (`ApplyConfigOverrides` in `internal/config/config.go`).
- If `tag_map_csv` is set in config, relative paths are resolved relative to the config file location (not CWD).
- Writes are disabled by default; enabling writes requires policy/env configuration (`MODBUS_WRITES_ENABLED=true` etc.).

## Test Quirks
- `modbus/rtu_integration_test.go` is env-guarded and skipped unless `MODBUS_RTU_TEST_PORT` is set (optional `MODBUS_RTU_TEST_BAUD`, `MODBUS_RTU_TEST_SLAVE_ID`).
- `modbus/soak_test.go` is opt-in and skipped unless `MODBUS_SOAK_TEST=1` (optional `MODBUS_SOAK_ITERATIONS`).
- `integration_transport_test.go` streamable test skips when local port `8080` is already in use.

## CI / Release Notes
- Release workflow runs `go test -v ./...` and `go vet ./...`, then cross-builds binaries for linux/darwin amd64+arm64.
- `go.mod` declares Go `1.25.0`, but CI and Dockerfiles are pinned to Go `1.23`; treat this version mismatch as a likely source of build/release failures.
- `Dockerfile` healthcheck probes `http://localhost:8081/health`, but runtime serves health on `:8080/health`; verify/fix this when touching container behavior.

## Local-Only Files
- `opencode.json` and `opencode.jsonc` point to a local MCP endpoint (`http://127.0.0.1:8080/mcp`); keep them local unless explicitly asked to version them.
