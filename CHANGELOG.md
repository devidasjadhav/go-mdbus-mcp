# Changelog

All notable changes to this project are documented in this file.

## v0.0.2 (2026-04-06)

Changes since `v0.0.1`:

### Added

- Added OpenCode usage and validation guide in `HOW_TO_USE.md`.
- Added streamable run helper target in `Makefile` (`run-streamable`).

### Changed

- Hardened Modbus connection handling to better tolerate idle timeout disconnects.
- Added configurable Modbus timeout and idle-timeout flags in `main.go`.
- Updated streamable HTTP setup to stateless JSON mode.
- Added graceful shutdown handling for streamable/SSE transports.
- Updated documentation to reflect resilient connection behavior.

### Fixed

- Fixed stale socket behavior that caused intermittent `EOF`/`broken pipe` after idle periods.
- Fixed single-register write path to use function code `0x06` when writing one register.
- Fixed tool input validation by rejecting zero read quantity and empty write arrays.
- Fixed default slave ID handling by consistently defaulting to `1` when omitted.

### Internal/Refactor

- Migrated codebase to official MCP Go SDK patterns.
- Simplified and cleaned legacy files/tests/docs from older structure.

### Commits

- `7214fed` fix: harden modbus reconnect flow and document opencode validation
- `c08e3b4` fix: default Slave ID to 1 and make it configurable in MCP tools
- `58199ea` docs: update README with new architecture and usage instructions
- `fcbaa8e` refactor: migrate to official MCP Go SDK
- `8fe4821` feat: add SSE transport support and connection pooling
- `1435dd4` feat: add stdio transport and apply DRY principles to tools
- `2ecb763` chore: remove bloat, dead code, and redundant binaries

## v0.0.1

- Initial tagged release.
