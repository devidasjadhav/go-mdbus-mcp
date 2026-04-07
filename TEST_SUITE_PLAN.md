# Test Suite and Benchmark Plan

Date: 2026-04-07

## Goals

- Validate correctness across all supported server transports: `stdio`, `sse`, `streamable`.
- Validate Modbus behavior across supported modes/drivers: `tcp` (`goburrow`, `simonvetter`) and `rtu` (env-gated hardware lane).
- Add staged complexity (basic -> advanced -> stress) with repeatable outputs.
- Produce comparison-ready benchmark reports (this server vs alternatives) in tabular form.

## Test Matrix

Primary matrix dimensions:

1. **MCP transport**: `stdio`, `sse`, `streamable`
2. **Modbus mode**: `tcp`, `rtu`
3. **Driver**: `goburrow`, `simonvetter`
4. **Workload profile**: basic, advanced, stress

Baseline required combinations for CI lane:

- `tcp + goburrow` over all 3 transports
- `tcp + simonvetter` over all 3 transports

Hardware/integration lane (nightly/manual):

- `rtu + simonvetter` over all 3 transports (requires `MODBUS_RTU_TEST_PORT`)

## Stage 1: Basic Functionality Suite

Purpose: verify core tool correctness and protocol wiring.

Scenarios:

- Server boot and graceful shutdown for each transport
- MCP handshake + tool discovery
- Read operations:
  - `read-coils`
  - `read-discrete-inputs`
  - `read-holding-registers`
  - `read-input-registers`
- Status operation:
  - `get-modbus-client-status`
- Negative input validation:
  - invalid address/quantity
  - missing required fields

Exit criteria:

- 100% pass on all baseline matrix combinations
- no panic/hang

## Stage 2: Advanced / Complex Behavior Suite

Purpose: validate policy, tags, retry/circuit, and edge behavior.

Scenarios:

- Write policy lanes:
  - writes disabled (expect guarded rejection)
  - writes enabled with allowlist (allow only allowed ranges)
- Write tools:
  - `write-holding-registers`
  - `write-coils`
- Tag workflows:
  - `read-tag`, `write-tag`
  - scaling, offset, data type encode/decode
  - tag slave-id override behavior
- Fault injection:
  - transient timeout/network reset to verify retry/backoff
  - repeated failures to verify circuit-open behavior and recovery
- Malformed payload resilience:
  - odd register byte lengths
  - short coil payloads

Exit criteria:

- all expected failures reported as MCP tool errors (not server crash)
- retry/circuit metrics visible via status tool and match expected transitions

## Stage 3: Stress and Performance Suite

Purpose: characterize throughput, latency, and stability under load.

Profiles:

1. **Read-heavy**: 95% reads, 5% status
2. **Mixed**: 70% reads, 30% writes (writes enabled policy lane)
3. **Write-sensitive**: serialized writes + concurrent reads

Load levels:

- concurrency: 1, 5, 10, 20, 50
- duration: 2m warmup + 5m measure per run
- repetitions: 3 per profile (report median + p95)

Captured metrics:

- requests/sec
- latency p50/p95/p99
- error rate (%)
- retries/sec
- circuit-open events
- process CPU%, RSS MB
- open sockets

Pass/fail guardrails (initial):

- error rate < 0.5% under read-heavy @ concurrency 20
- no memory growth trend > 10% over 10m steady-state run
- no goroutine leak after test teardown

## Comparison Framework (This Server vs Others)

Comparison dimensions:

- correctness pass rate (%)
- feature parity score (tool support + tag/policy features)
- throughput (RPS)
- tail latency (p95/p99)
- resilience score (retry/circuit behavior under fault injection)
- resource efficiency (CPU/RSS per 1k req)

Suggested competitors/adapters:

- adapter lane for alternate Modbus MCP servers
- if no MCP-native competitor, compare via equivalent Modbus operation harness

## Report Format (Tabular)

### Table A: Correctness Summary

| Server | Transport | Driver/Mode | Stage | Total | Passed | Failed | Pass % |
|---|---|---|---:|---:|---:|---:|---:|

### Table B: Performance Summary

| Server | Profile | Concurrency | RPS | p50 ms | p95 ms | p99 ms | Error % | CPU % | RSS MB |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|

### Table C: Reliability Under Faults

| Server | Fault Type | Retries Triggered | Circuit Opens | Recovery Time ms | Final Pass |
|---|---|---:|---:|---:|---|

## Implementation Plan

1. Add `tests/e2e` harness with transport-aware runner and reusable MCP client helpers.
2. Add `tests/fault` lane with controllable mock/fault proxy.
3. Add `tests/stress` runner (configurable concurrency/profile/duration).
4. Emit machine-readable artifacts (`json` + `csv`) for all runs.
5. Add report generator to produce markdown tables from artifacts.
6. Wire CI lanes:
   - PR: Stage 1 + selected Stage 2
   - Nightly: full Stage 2 + Stage 3 + RTU lane (if hardware available)

## Immediate Next Steps

1. Implement Stage 1 harness first (fastest ROI).
2. Add artifact schema and report generation early so later stages are plug-in.
3. Add one baseline stress profile (`read-heavy`, concurrency 10) before expanding matrix.
