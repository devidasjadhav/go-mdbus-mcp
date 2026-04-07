package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type correctnessRow struct {
	Server    string
	Transport string
	Driver    string
	Stage     string
	Passed    bool
	Duration  string
	Error     string
}

type stage3Row struct {
	Server       string
	Transport    string
	Driver       string
	Profile      string
	Concurrency  int
	RPS          float64
	P50Ms        float64
	P95Ms        float64
	P99Ms        float64
	ErrorRatePct float64
	RetriesDelta uint64
	CircuitOpen  bool
	Passed       bool
	Error        string
}

func main() {
	server := flag.String("server", "go-mdbus-mcp", "Server name label for the report")
	stage1 := flag.String("stage1", "", "Path to stage1-results.json")
	stage2 := flag.String("stage2", "", "Path to stage2-results.json")
	stage3 := flag.String("stage3", "", "Path to stage3-results.json")
	out := flag.String("out", "", "Output markdown file path (stdout if empty)")
	flag.Parse()

	cRows := make([]correctnessRow, 0)
	if strings.TrimSpace(*stage1) != "" {
		rows, err := loadCorrectness(*stage1, *server, "stage1")
		if err != nil {
			fatalf("load stage1: %v", err)
		}
		cRows = append(cRows, rows...)
	}
	if strings.TrimSpace(*stage2) != "" {
		rows, err := loadCorrectness(*stage2, *server, "stage2")
		if err != nil {
			fatalf("load stage2: %v", err)
		}
		cRows = append(cRows, rows...)
	}

	pRows := make([]stage3Row, 0)
	if strings.TrimSpace(*stage3) != "" {
		rows, err := loadStage3(*stage3, *server)
		if err != nil {
			fatalf("load stage3: %v", err)
		}
		pRows = rows
	}

	report := buildMarkdownReport(*server, cRows, pRows)
	if strings.TrimSpace(*out) == "" {
		_, _ = os.Stdout.WriteString(report)
		return
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fatalf("create output dir: %v", err)
	}
	if err := os.WriteFile(*out, []byte(report), 0o644); err != nil {
		fatalf("write report: %v", err)
	}
}

func loadCorrectness(path, server, stage string) ([]correctnessRow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw []map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	rows := make([]correctnessRow, 0, len(raw))
	for _, r := range raw {
		rows = append(rows, correctnessRow{
			Server:    server,
			Transport: asString(r["transport"]),
			Driver:    asString(r["driver"]),
			Stage:     stage,
			Passed:    asBool(r["passed"]),
			Duration:  asString(r["duration"]),
			Error:     asString(r["error"]),
		})
	}
	return rows, nil
}

func loadStage3(path, server string) ([]stage3Row, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw []map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	rows := make([]stage3Row, 0, len(raw))
	for _, r := range raw {
		rows = append(rows, stage3Row{
			Server:       server,
			Transport:    asString(r["transport"]),
			Driver:       asString(r["driver"]),
			Profile:      asString(r["profile"]),
			Concurrency:  asInt(r["concurrency"]),
			RPS:          asFloat(r["rps"]),
			P50Ms:        asFloat(r["p50_ms"]),
			P95Ms:        asFloat(r["p95_ms"]),
			P99Ms:        asFloat(r["p99_ms"]),
			ErrorRatePct: asFloat(r["error_rate_pct"]),
			RetriesDelta: asUint(r["retries_delta"]),
			CircuitOpen:  asBool(r["circuit_open"]),
			Passed:       asBool(r["passed"]),
			Error:        asString(r["error"]),
		})
	}
	return rows, nil
}

func buildMarkdownReport(server string, cRows []correctnessRow, pRows []stage3Row) string {
	sort.Slice(cRows, func(i, j int) bool {
		a, b := cRows[i], cRows[j]
		if a.Stage != b.Stage {
			return a.Stage < b.Stage
		}
		if a.Transport != b.Transport {
			return a.Transport < b.Transport
		}
		return a.Driver < b.Driver
	})
	sort.Slice(pRows, func(i, j int) bool {
		a, b := pRows[i], pRows[j]
		if a.Profile != b.Profile {
			return a.Profile < b.Profile
		}
		if a.Concurrency != b.Concurrency {
			return a.Concurrency < b.Concurrency
		}
		if a.Transport != b.Transport {
			return a.Transport < b.Transport
		}
		return a.Driver < b.Driver
	})

	var sb strings.Builder
	sb.WriteString("# Test Comparison Report\n\n")
	sb.WriteString(fmt.Sprintf("Server: `%s`\n\n", server))

	sb.WriteString("## Table A: Correctness Summary\n\n")
	sb.WriteString("| Server | Transport | Driver/Mode | Stage | Total | Passed | Failed | Pass % | Notes |\n")
	sb.WriteString("|---|---|---|---:|---:|---:|---:|---:|---|\n")
	if len(cRows) == 0 {
		sb.WriteString("| - | - | - | - | 0 | 0 | 0 | 0.00 | no data |\n")
	} else {
		type key struct{ stage, transport, driver string }
		agg := map[key]struct {
			total, pass, fail int
			notes             []string
		}{}
		for _, r := range cRows {
			k := key{r.Stage, r.Transport, r.Driver}
			v := agg[k]
			v.total++
			if r.Passed {
				v.pass++
			} else {
				v.fail++
				if strings.TrimSpace(r.Error) != "" {
					v.notes = append(v.notes, r.Error)
				}
			}
			agg[k] = v
		}
		keys := make([]key, 0, len(agg))
		for k := range agg {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].stage != keys[j].stage {
				return keys[i].stage < keys[j].stage
			}
			if keys[i].transport != keys[j].transport {
				return keys[i].transport < keys[j].transport
			}
			return keys[i].driver < keys[j].driver
		})
		for _, k := range keys {
			v := agg[k]
			passPct := float64(v.pass) / float64(v.total) * 100
			notes := "ok"
			if len(v.notes) > 0 {
				notes = truncate(strings.Join(v.notes, " | "), 90)
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d | %d | %d | %.2f | %s |\n", server, k.transport, k.driver, k.stage, v.total, v.pass, v.fail, passPct, sanitizeCell(notes)))
		}
	}

	sb.WriteString("\n## Table B: Performance Summary\n\n")
	sb.WriteString("| Server | Profile | Concurrency | Transport | Driver | RPS | p50 ms | p95 ms | p99 ms | Error % |\n")
	sb.WriteString("|---|---|---:|---|---|---:|---:|---:|---:|---:|\n")
	if len(pRows) == 0 {
		sb.WriteString("| - | - | 0 | - | - | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |\n")
	} else {
		for _, r := range pRows {
			sb.WriteString(fmt.Sprintf("| %s | %s | %d | %s | %s | %.3f | %.3f | %.3f | %.3f | %.3f |\n",
				r.Server, r.Profile, r.Concurrency, r.Transport, r.Driver, r.RPS, r.P50Ms, r.P95Ms, r.P99Ms, r.ErrorRatePct))
		}
	}

	sb.WriteString("\n## Table C: Reliability Under Faults\n\n")
	sb.WriteString("| Server | Transport | Driver | Profile | Retries Triggered | Circuit Opens | Final Pass | Notes |\n")
	sb.WriteString("|---|---|---|---|---:|---|---|---|\n")
	if len(pRows) == 0 {
		sb.WriteString("| - | - | - | - | 0 | false | false | no data |\n")
	} else {
		for _, r := range pRows {
			notes := "ok"
			if !r.Passed && strings.TrimSpace(r.Error) != "" {
				notes = truncate(r.Error, 90)
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d | %t | %t | %s |\n",
				r.Server, r.Transport, r.Driver, r.Profile, r.RetriesDelta, r.CircuitOpen, r.Passed, sanitizeCell(notes)))
		}
	}

	return sb.String()
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asBool(v any) bool {
	b, ok := v.(bool)
	if ok {
		return b
	}
	return false
}

func asFloat(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

func asInt(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}

func asUint(v any) uint64 {
	if f, ok := v.(float64); ok {
		if f < 0 {
			return 0
		}
		return uint64(f)
	}
	return 0
}

func sanitizeCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "/")
	return strings.TrimSpace(s)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
