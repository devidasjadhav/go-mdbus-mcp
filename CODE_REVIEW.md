# Code Review Report

Date: 2026-04-07

## Scope

Deep review of logic, architecture, performance, and design principles across:

- runtime bootstrap (`main.go`, `internal/mcpserver`)
- config and policy (`internal/config`, `modbus/write_policy.go`)
- drivers and retries (`modbus/client*.go`, `modbus/factory.go`)
- tool layer (`modbus/tool_*.go`)
- tests, CI, Docker, and documentation consistency

## Current Quality Snapshot

- Tests: `go test -v ./...` passes (RTU + soak tests are env-gated and skipped by default).
- Static checks: `go vet ./...` passes.
- Statement coverage (from `coverage.out`): **32.04%** (404/1261 statements).
- Exported-code doc-comment coverage (static estimate): **24.68%** (19/77 exported items).

## Priority Findings

### P0 (Correctness / Delivery Risk)

1. **Container healthcheck points to wrong port**
   - Runtime serves `/health` on `:8080`, Docker probes `:8081`.
   - Files: `Dockerfile`, `internal/mcpserver/runtime.go`.

2. **Toolchain mismatch across module vs CI/container**
   - `go.mod` requires `1.25.0`, but release workflow and Docker use `1.23`.
   - Files: `go.mod`, `.github/workflows/release.yml`, `Dockerfile`.

3. **Low effective coverage in critical MCP tool path**
   - `main.go`, `tool_registers.go`, `tool_tags.go`, `tool_helpers.go`, `tool_status.go` report 0% under current method.
   - Most user-visible logic sits here.

### P1 (Runtime Safety / Robustness)

1. **Potential panic: missing nil guard in driver factory**
   - `NewDriver(config *Config)` dereferences `config` without nil check.
   - File: `modbus/factory.go`.

2. **Potential panic: coil write path lacks input-length guard**
   - `simonvetter` coil writer indexes packed-byte payload without validating size.
   - File: `modbus/client_simonvetter.go`.

3. **Potential panic on malformed register byte responses**
   - Some decode loops assume even byte count and index `i+1`.
   - Files: `modbus/tool_registers.go`, `modbus/tool_tags.go`.

4. **Lenient env bool parse can hide misconfiguration**
   - Invalid bool values silently fall back.
   - File: `modbus/write_policy.go`.

### P2 (Maintainability / Performance)

1. **Throughput bottleneck by coarse locking**
   - Driver `Execute` serializes operations with one mutex; concurrent tool calls become single-filed.
   - Files: `modbus/client.go`, `modbus/client_simonvetter.go`.

2. **Aggressive reconnect-per-op tradeoff**
   - Safe for flaky peers, but expensive for stable links/high-rate polling.
   - File: `modbus/client.go`.

3. **Missing HTTP server hardening timeouts**
   - No explicit `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` on HTTP server.
   - File: `internal/mcpserver/runtime.go`.

4. **DRY duplication in byte/coil conversion logic**
   - Similar pack/unpack and word decode logic repeated in multiple files.
   - Files: `modbus/tool_registers.go`, `modbus/tool_tags.go`, `modbus/client_simonvetter.go`, `modbus/mock_client.go`.

## Principles Assessment

### SOLID

- **S (mostly good):** clean package boundaries and separation between config/runtime/drivers/tools.
- **O (partial):** adding a new driver still requires editing `switch` in factory (`modbus/factory.go`).
- **L (good):** tool layer depends on `Driver` abstraction.
- **I (acceptable):** `Driver` interface is broad but still manageable.
- **D (good):** high-level handlers depend on interface, not concrete client types.

### DRY

- Byte conversion and coil bit-packing logic is duplicated; should be centralized to shared helpers.

### KISS

- `main.go` is still too busy (flag parsing + config merge + warnings + driver/server bootstrap).

### Separation of Concerns

- Generally good at package level; minor overlap remains in startup orchestration and repeated transform code in handlers.

### YAGNI

- Several large planning/report docs appear historical and not runtime-critical (see stale docs section).

## Coverage Detail

### Test Coverage

- Overall statement coverage: **32.04%**.
- Better-covered areas:
  - `modbus/write_policy.go` ~71.6%
  - `internal/config/config.go` ~58.5%
  - `modbus/mock_client.go` ~58.1%
  - `modbus/tag_codec.go` ~53.2%
  - `modbus/tag_map.go` ~54.3%
- Under-covered areas:
  - `main.go` 0%
  - `modbus/tool_registers.go` 0%
  - `modbus/tool_tags.go` 0%
  - `modbus/tool_helpers.go` 0%
  - `modbus/tool_status.go` 0%
  - `modbus/client_simonvetter.go` ~8.9%

### Documentation Coverage

- Exported symbol doc-comment coverage: **24.68%** (19/77).
- Most missing docs are in:
  - `internal/config/config.go`
  - `modbus/client.go`
  - `modbus/client_simonvetter.go`
  - `modbus/tag_map.go`
  - `modbus/tool_types.go`

## Stale / Optional Files Review

Candidates to archive or remove if not actively maintained:

1. `docs/archive/PLAN.md` (historical roadmap with outdated checklist state)
2. `docs/archive/Tasks.md` (completed migration execution log)
3. `docs/archive/TEST_DOCUMENT.md` (manual validation log, high churn)

Potentially optional (keep only if product/comms needs them):

4. `COMPETITIVE_COMPARISON_REPORT.md`
5. `CODE_REVIEW.md` (this file should be treated as living doc; if not maintained, archive it)

## Coherent Step-by-Step Fix Plan (Clustered Passes)

The plan below intentionally groups related changes so each pass is coherent, minimizes rebasing churn, and avoids touching the same hotspots repeatedly.

### Pass 1: Baseline Consistency and Deployment Safety

Goal: eliminate immediate release/runtime mismatch risk.

Changes:

1. Align healthcheck to `8080`.
2. Align Go version strategy across `go.mod`, CI, and Docker (pick one supported version and apply consistently).
3. Add a short "toolchain policy" note in docs (`README.md` or `AGENTS.md`) to prevent drift recurrence.

Files likely touched:

- `Dockerfile`
- `.github/workflows/release.yml`
- `go.mod` (only if version decision changes)
- `README.md` and/or `AGENTS.md`

Verification:

- `go test -v ./...`
- `go vet ./...`
- Docker build + local health probe smoke test

### Pass 2: Runtime Guardrails and Panic-Proofing

Goal: harden input/path safety without large refactors.

Changes:

1. Add nil-config guard in `NewDriver`.
2. Validate packed coil payload length before indexing in simonvetter write path.
3. Add defensive even-length checks before `uint16` decode loops.
4. Make env bool parse strict for critical safety variables (return config error on invalid bool).

Files likely touched:

- `modbus/factory.go`
- `modbus/client_simonvetter.go`
- `modbus/tool_registers.go`
- `modbus/tool_tags.go`
- `modbus/write_policy.go`

Verification:

- Add/extend unit tests for malformed input paths.
- `go test -v ./modbus -run "(Write|Read|Policy|Factory)"`
- `go test -v ./...`

### Pass 3: Tool-Layer Testability and Coverage Lift

Goal: increase confidence where user-facing behavior lives.

Changes:

1. Add in-process unit tests for tool handlers (table-driven cases):
   - read/write register tools
   - read/write tag tools
   - status tool and helper error mapping
2. Introduce test doubles for `Driver` to cover success/failure/edge paths cheaply.
3. Keep integration tests for protocol smoke, but move logic assertions to in-process tests for coverage.

Files likely touched:

- new tests under `modbus/*_test.go`
- possibly lightweight test helper file in `modbus`

Verification:

- `go test -cover ./...`
- coverage trend report in PR notes

Target after pass:

- overall coverage >= 50%
- tool files no longer at 0%

### Pass 4: DRY Refactor (Single Conversion Utility Layer)

Goal: remove repeated bit/word conversion code and reduce bug surface.

Changes:

1. Extract shared helpers for:
   - bytes -> words
   - words -> bytes
   - bool slice <-> packed coil bytes
2. Replace duplicated implementations across tools, mock client, and simonvetter adapter.
3. Keep behavior unchanged; pure refactor with tests guarding parity.

Files likely touched:

- `modbus/tool_registers.go`
- `modbus/tool_tags.go`
- `modbus/client_simonvetter.go`
- `modbus/mock_client.go`
- new helper file in `modbus`

Verification:

- `go test -v ./modbus`
- benchmark sanity check `go test -bench=. -benchmem ./modbus`

### Pass 5: Runtime Throughput and HTTP Hardening

Goal: improve operational behavior under load/network pressure.

Changes:

1. Add explicit HTTP server timeouts in `internal/mcpserver/runtime.go`.
2. Revisit lock granularity/connection lifecycle strategy:
   - keep safety guarantees
   - reduce serialization where possible
3. Make reconnect policy configurable per transport/driver profile (safe defaults remain).

Files likely touched:

- `internal/mcpserver/runtime.go`
- `modbus/client.go`
- `modbus/client_simonvetter.go`
- config docs/tests as needed

Verification:

- existing tests
- targeted stress/parallel tool-call test
- benchmarks before/after

Status note (2026-04-07): HTTP timeout hardening is implemented. Driver lock-granularity and reconnect-strategy refactor is intentionally deferred to a dedicated follow-up pass to avoid mixing behavior-changing concurrency work with safety/config updates.

### Pass 6: Documentation and Repo Hygiene

Goal: keep docs accurate and reduce noise.

Changes:

1. Raise exported symbol doc-comment coverage on key APIs (`internal/config`, driver APIs, tag map, write policy).
2. Resolve doc inconsistencies (`ARCHITECTURE.md`, compatibility wording, tooling notes).
3. Archive or remove stale planning artifacts (`docs/archive/PLAN.md`, `docs/archive/Tasks.md`, `docs/archive/TEST_DOCUMENT.md`) if team agrees.

Verification:

- run docs lint/check if available
- quick link/file consistency pass

## Suggested Milestone Acceptance Criteria

- No known P0 findings remain.
- Coverage >= 50% overall and >= 60% in `modbus` package.
- No tool-handler file remains at 0%.
- CI + Docker + local toolchain version policy is consistent and documented.
- Stale docs either archived or clearly marked as historical.
