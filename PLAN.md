# Production Improvement Plan

This plan upgrades `go-mdbus-mcp` from a fast protocol bridge to a production-safe industrial MCP server.

## Goals

1. Enforce safe write behavior by default.
2. Improve resilience for flaky industrial networks.
3. Add semantic abstractions (tags and data types) for reliable LLM use.
4. Add testability and benchmarks for repeatable quality.
5. Keep runtime lightweight for edge Linux/OpenBMC deployments.

## Current Gaps

- No explicit environment-driven write safety gate.
- Partial reconnect behavior, but no formal retry/circuit policy.
- Raw register-level interface only (limited semantic context for LLMs).
- No built-in mock provider for deterministic CI and local validation.
- No formal benchmark harness and SLO baseline.

## Architecture Roadmap

### Phase 1: Safety Guardrails (Write Gating)

Scope:

- Add env-driven write gate:
  - `MODBUS_WRITES_ENABLED=false` (default deny)
  - `MODBUS_WRITE_ALLOWLIST` (optional address ranges)
  - `MODBUS_WRITE_REQUIRE_CONFIRMATION` (optional second-step protection)
- Apply to all mutating tools:
  - `write-holding-registers`
  - `write-coils`
  - future `write-tag`
- Return structured guarded rejection errors with clear remediation text.

Implementation notes:

- Introduce `WritePolicy` struct loaded at startup.
- Validate writes before any Modbus frame is sent.
- Add explicit audit logs for allowed and blocked writes.

Acceptance criteria:

- Writes are blocked by default.
- Writes succeed only when policy allows.
- Rejections are deterministic and machine-readable.

---

### Phase 2: Connection Lifecycle Hardening

Scope:

- Add retry policy for transient errors (`EOF`, timeout, broken pipe).
- Add bounded exponential backoff (short and deterministic).
- Add request-level timeout and cancellation propagation.
- Add simple circuit-breaker/cooldown after repeated failures.
- Extend status reporting with health counters.

Implementation notes:

- Introduce `RetryPolicy` in Modbus config.
- Retry only idempotent operations by default (reads).
- For writes, allow `retry_on_write` as explicit opt-in.
- Ensure context cancellation stops retries immediately.

Acceptance criteria:

- Short network drops recover without restarting MCP server.
- Failure storms do not lock worker goroutines.
- Status endpoint/tools expose recent failures and retry counts.

---

### Phase 3: Tag Mapping Layer

Scope:

- Add config-driven tag map (`yaml` or `json`).
- Define tag schema fields:
  - `name`, `address`, `function`, `quantity`, `type`, `word_order`, `byte_order`, `scale`, `offset`, `access`
- New tools:
  - `list-tags`
  - `read-tag`
  - `write-tag` (guarded)

Implementation notes:

- Load and validate map at startup.
- Keep O(1) lookup by tag name.
- Preserve raw register tools for advanced users.

Acceptance criteria:

- AI can read and write tags by semantic names.
- Invalid tag config fails fast at startup with clear errors.

---

### Phase 4: Data Types and Codec Engine

Scope:

- Add typed decode/encode support:
  - `uint16/int16`
  - `uint32/int32`
  - `float32`
  - optional `uint64/int64`
- Add endian controls and word swap handling.
- Add scale/offset transforms per tag.

Implementation notes:

- Build codec helpers in a dedicated package (`modbus/codec`).
- Add strict validation: quantity/type compatibility.

Acceptance criteria:

- Typed read/write round-trips match expected values.
- Endianness behavior is deterministic and documented.

---

### Phase 5: TCP Proxy Debugger Maturation

Scope:

- Keep current proxy tools and extend with:
  - request/response correlation by transaction ID
  - latency calculation per transaction
  - anomaly classification (length mismatch, exception responses, parse errors)
  - export capture (`jsonl`) for offline analysis
- Add filter extensions:
  - function code, unit ID, connection ID, direction, errors-only (already partially implemented in WIP)

Acceptance criteria:

- Protocol-level fault diagnosis is possible via MCP tools alone.
- Captures are bounded and production-safe.

---

### Phase 6: Mocking and Test Harness

Scope:

- Introduce provider interface abstraction:
  - real Modbus provider
  - mock provider
- Add mock server mode for local and CI.
- Add scenario tests:
  - normal reads/writes
  - invalid responses
  - dropped connections
  - restart recovery

Acceptance criteria:

- CI validates core flows without physical hardware.
- Deterministic failure scenarios are reproducible.

---

### Phase 7: Benchmark and SLO Suite

Scope:

- Add benchmark harness for:
  - heartbeat/ping
  - read (small and medium payloads)
  - write + verify
  - burst reads (50+)
  - concurrent agent access
  - restart recovery scenario
- Publish median and p95 latency baseline.

Acceptance criteria:

- Reproducible benchmark command exists in repo.
- Baseline results documented and tracked across releases.

## Security and Operations

- Default docs/examples to non-root ports (`1502` for proxy/mock).
- Warn on privileged bind (`<1024`) and root runtime.
- Document safe deployment profile:
  - dedicated service user
  - restricted network ACL
  - minimal container runtime permissions
- Add `SECURITY.md` with threat model and write-safety controls.

## Implementation Order (Recommended)

1. Phase 1: Write Gating
2. Phase 2: Lifecycle Hardening
3. Phase 3 + 4: Tag Mapping + Data Types
4. Phase 6: Mocking and CI scenarios
5. Phase 5: Proxy maturation
6. Phase 7: Benchmarks and SLO docs

## Deliverables Checklist

- [ ] Write policy env vars and enforcement
- [ ] Retry/backoff/circuit controls with status counters
- [ ] Tag map schema and new tag tools
- [ ] Data type codec engine with tests
- [ ] Mock provider and deterministic scenario tests
- [ ] Proxy correlation and export
- [ ] Benchmark runner and documented baseline
- [ ] Security and operations documentation
