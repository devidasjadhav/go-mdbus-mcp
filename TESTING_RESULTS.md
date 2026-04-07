# Testing Results and Methodology

Date: 2026-04-07

## What We Test

This repository now has a staged test strategy:

1. Stage 1: basic MCP transport and core read/status tool behavior
2. Stage 2: advanced behavior (tags, write-policy, negative paths)
3. Stage 3: stress/performance (RPS, p50/p95/p99, error rate)
4. External comparison: this server vs other Modbus MCP servers on a shared backend

Reference test plan: `TEST_SUITE_PLAN.md`

## How We Run Tests

## Internal staged suite

Use Make targets:

```bash
make stage1
make stage2
make stress-quick
make stress STAGE3_DURATION=3s STAGE3_CONCURRENCY=1,5,10
make report
```

Artifact outputs (JSON/CSV + markdown report) are written to:

- `ARTIFACT_DIR` (default `/tmp/modbus-mcp-artifacts`)
- report path `REPORT_PATH` (default `$(ARTIFACT_DIR)/comparison-report.md`)

## External server comparison

Comparison runner:

```bash
MODBUS_BENCH_HOST=127.0.0.1 MODBUS_BENCH_PORT=5002 go run ./tools/extcompare
```

Output:

- `/tmp/mcp-servers-Uonh4R/compare-results.json`

## Environment used for comparison

- Shared backend: `mbserver` on `127.0.0.1:5002`
- Compared servers:
  - `go-mdbus-mcp`
  - `kukapay/modbus-mcp`
  - `alejoseb/ModbusMCP`
  - `midhunxavier/MODBUS-MCP`
  - `ezhuk/modbus-mcp`

Notes:

- `ezhuk` requires Python >= 3.13. Test runs used local Python 3.14 build at `/home/dev/Downloads/python/Python-3.14.3/python` with a dedicated venv.
- Transport support differs by project; not every server supports stdio+sse+streamable.

## Internal Staged Suite Results

From run artifacts at `/tmp/driver-compare-Ist8pj`:

- Stage 1: pass across `stdio`, `sse`, `streamable` with both drivers (`goburrow`, `simonvetter`)
- Stage 2: pass across same matrix, including policy/tag negative and positive paths
- Stage 3: pass with 0 retry/circuit events in mock-mode stress runs

Selected Stage 3 performance examples (from generated report):

- `sse`, `mixed`, conc=5:
  - goburrow: `5223.0 RPS`, p95 `1.698 ms`
  - simonvetter: `5558.7 RPS`, p95 `1.544 ms`
- `streamable`, `read-heavy`, conc=5:
  - goburrow: `4946.3 RPS`, p95 `1.635 ms`
  - simonvetter: `4287.0 RPS`, p95 `2.055 ms`

Conclusion: both drivers are stable; winner depends on transport/profile.

## External Comparison Results (shared backend: mbserver:5002)

Fair run summary (same backend, read-tool stress lane, 0% errors):

| Server | RPS (conc=1) | RPS (conc=5) | p95 ms (conc=5) |
|---|---:|---:|---:|
| go-mdbus-mcp | 753.9 | 804.9 | 6.96 |
| alejoseb/ModbusMCP | 227.7 | 781.4 | 6.38 |
| kukapay/modbus-mcp | 127.5 | 437.3 | 8.03 |
| midhunxavier/MODBUS-MCP | 141.1 | 426.4 | 8.12 |
| ezhuk/modbus-mcp | 9.4 | 45.1 | 115.92 |

Observations:

- `go-mdbus-mcp` led throughput overall in this backend run.
- `alejoseb` was close at conc=5 with slightly lower p95.
- `ezhuk` was functional but significantly slower on this setup.

## Runtime Tuning Results (actual backend)

Matrix was run over:

- drivers: `goburrow`, `simonvetter`
- pool sizes: `1,2,4`
- reconnect per operation: `true,false`
- concurrencies: `5,10`

Results file:

- `/tmp/tuning-results-correct.json`

Best observed config in this environment:

- `--modbus-driver simonvetter`
- `--modbus-connection-pool-size 2`
- `--modbus-reconnect-per-operation=false`

At conc=10: `758.7 RPS`, p95 `21.66 ms`, error `0%`

Safe baseline remained close:

- `goburrow`, pool `1`, reconnect `true`
- conc=10: `744.7 RPS`, p95 `22.62 ms`, error `0%`

Conclusion: tuning helps, but gains were modest on this backend.

## Failures We Investigated and Root Cause

1. `function 132` / Modbus exception `0x84` (`Read Input Registers` exception)
   - Root cause was backend/simulator behavior mismatch, not missing support in this server.
   - Verified healthy behavior against `mbserver:5002`.

2. Intermittent benchmark "endpoint busy" or wrong transport behavior
   - Root cause was process lifecycle in ad-hoc harness runs (`go run` child process left behind on port 8080).
   - Mitigation: explicit cleanup and endpoint checks before each target run.
