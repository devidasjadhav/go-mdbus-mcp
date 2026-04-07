# Modbus MCP Competitive Comparison

Date: 2026-04-07

Scope compared:

- `devidasjadhav/go-mdbus-mcp` (this repo)
- `kukapay/modbus-mcp`
- `alejoseb/ModbusMCP`
- `midhunxavier/MODBUS-MCP`
- `ezhuk/modbus-mcp`
- `KerberosClaw/kc_modbus_mcp`

Method used:

- Reviewed each repository README (public source of intended feature set).
- Reviewed GitHub repository metadata (language, stars, recency).
- For this repo, used current local code state as source of truth (not only README).

---

## Executive Summary

Your server is currently strongest in:

1. **Go runtime simplicity + small footprint** (good for industrial deployment).
2. **Transport flexibility in one binary** (`stdio`, `sse`, `streamable`).
3. **Straightforward operational model** for a single target Modbus TCP endpoint.

Competitors are generally stronger in:

1. **Breadth of Modbus operations** (discrete inputs/input registers, server/slave simulation, advanced function codes).
2. **Semantic layer** (tag maps/device profiles and typed conversions).
3. **Testing maturity and ecosystem docs** (especially Python/FastMCP projects).

Bottom line: your project is a clean, practical core. To lead this space, prioritize **feature depth + safety controls + test coverage**.

---

## Comparison Table

| Repo | Stack | MCP transport model | Modbus scope | Semantic layer | Test/deploy maturity | Notable strengths | Main gaps vs others |
|---|---|---|---|---|---|---|---|
| **devidasjadhav/go-mdbus-mcp** | Go + official `go-sdk` | `stdio`, `sse`, `streamable` | Holding regs + coils (read/write) over TCP | No tag map/profile in current code | Basic | Minimal binary, easy OpenCode/Claude integration, low runtime overhead | Missing discrete/input registers, no RTU/UDP, limited tool breadth, light tests |
| kukapay/modbus-mcp | Python | Primarily stdio MCP | Coils + holding + input registers + multiple helpers | Prompt-level analysis | Basic-medium | Multi-transport Modbus support claim (`tcp/udp/serial`) and broader tool list | Less evidence of deep reliability/testing compared to larger peers |
| alejoseb/ModbusMCP | Python + FastMCP + pymodbus | Stdio-oriented MCP usage | Very broad: client + server(slave), full register families | Connection/server management abstractions | Medium-high (README shows broad verification matrix) | Most complete operational surface (including server lifecycle/datastore tools) | Python runtime overhead; larger operational complexity |
| midhunxavier/MODBUS-MCP | Python + Node (dual implementation) | Stdio MCP | Broad read/write + typed decode/encode + retries/chunking | Tag map support (`read_tag/write_tag`) | Medium | Dual runtime options + typed conversions + reliability knobs | Project complexity split across subprojects; maintenance depth unclear |
| ezhuk/modbus-mcp | Python + FastMCP 2 | Streamable HTTP first-class (also CLI) | Register read/write focused, production-oriented docs | Device config and multi-device patterns | Medium-high | Strong modern MCP posture (HTTP streamable), examples, Docker, auth notes | Python dependency chain; some advanced Modbus functions not emphasized |
| KerberosClaw/kc_modbus_mcp | Python + FastMCP 3 | Streamable HTTP | Profile-mode + raw mode, typed conversion | Strong YAML device profiles, built-in simulator | Medium | Best semantic UX (named registers), simulator included, practical demoability | Early-stage; fewer stars/history; RTU listed as TODO |

---

## Detailed Analysis

## 1) Protocol and MCP alignment

- Your server now aligns well at the transport layer: supports legacy (`stdio`, `sse`) and current (`streamable HTTP`) patterns.
- `ezhuk` and `KerberosClaw` are strongly HTTP/streamable-centric, which matches modern remote MCP deployment patterns.
- `alejoseb` appears strongest for comprehensive operations but is more focused on stdio/IDE workflows.

Assessment: **you are competitive on MCP transport support**.

## 2) Modbus capability depth

Current local code in this repo exposes four core tools:

- `read-holding-registers`
- `read-coils`
- `write-holding-registers`
- `write-coils`

with optional `slave_id`.

Compared with peers:

- Multiple competitors support all 4 Modbus data areas (coils, discrete inputs, holding, input registers).
- Several support server/slave simulation and lifecycle tools (`alejoseb`, `KerberosClaw` simulator).
- Some provide advanced operations (typed decoding, device identification, masked writes).

Assessment: **this is the biggest competitive gap**.

## 3) Safety, reliability, and operability

Your code has useful baseline reliability:

- Mutex-guarded Modbus execution.
- Per-call connection usage and error mapping.
- Configurable slave id in tool args.

Peers add more operational controls:

- Retry/backoff knobs and tool timeouts (`midhunxavier` project docs).
- Broader troubleshooting docs and server management workflows (`alejoseb`).
- Packaged container docs and examples (`ezhuk`, `KerberosClaw`).

Assessment: **good baseline, but missing policy/safety features for production controls**.

## 4) Developer experience and adoption signals

- Your project: recent activity, low stars, clean scope.
- `kukapay`: higher stars, but less recent code push.
- `ezhuk`: active recently with packaged distribution and docs.
- Others: lower stars but richer feature sets in niche use-cases.

Assessment: **adoption will depend on shipping deeper capabilities + proving reliability**.

---

## Where this repo is better today

- **Lean deployment story**: one Go binary, minimal external runtime requirements.
- **Transport optionality in one place**: can serve local stdio clients and remote streamable clients without changing project.
- **Lower cognitive load**: easy to reason about and debug compared to very broad multi-mode projects.

---

## Where this repo is behind today

1. Missing Modbus function coverage (discrete inputs, input registers, richer operations).
2. Missing semantic abstraction (tag map / named points / profile layer).
3. No built-in simulator/mock mode for zero-hardware validation.
4. Limited automated tests after SDK migration (especially transport and tool contract tests).
5. No explicit write-policy/allowlist guardrails in active code.

---

## Recommended Roadmap (Priority Order)

## Phase 1 — Close capability gaps fast

1. Add `read-discrete-inputs` and `read-input-registers` tools.
2. Add optional typed decoding helpers for holding/input registers (`int16/int32/float32`, byte/word order).
3. Add write safeguards (`MODBUS_WRITES_ENABLED`, address allowlists) in active code paths.

## Phase 2 — Improve reliability and trust

1. Reintroduce robust integration tests (stdio + streamable tool calls).
2. Add fault-injection tests for timeout/reconnect/slave mismatch.
3. Add metrics/status tool (connection errors, retry counts, last success time).

## Phase 3 — Differentiate

1. Add semantic tag map (`list-tags/read-tag/write-tag`) with CSV/JSON source.
2. Add built-in mock simulator mode for demos and CI.
3. Add optional RTU support if target users are shop-floor serial networks.

---

## Practical Positioning Statement

If you keep this repo lean and Go-native, the best market position is:

> **“Production-friendly Modbus MCP for operators who want a small, secure, streamable-first binary with predictable behavior.”**

To make that credible against peers, add **full Modbus read surface + safety controls + transport contract tests**.

---

## Sources

- https://github.com/devidasjadhav/go-mdbus-mcp
- https://github.com/kukapay/modbus-mcp
- https://github.com/alejoseb/ModbusMCP
- https://github.com/midhunxavier/MODBUS-MCP
- https://github.com/ezhuk/modbus-mcp/
- https://github.com/KerberosClaw/kc_modbus_mcp
