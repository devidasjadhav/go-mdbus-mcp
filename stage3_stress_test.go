package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type stressProfile string

const (
	profileReadHeavy stressProfile = "read-heavy"
	profileMixed     stressProfile = "mixed"
	profileWriteFav  stressProfile = "write-sensitive"
)

type stage3Result struct {
	Transport       string  `json:"transport"`
	Driver          string  `json:"driver"`
	Profile         string  `json:"profile"`
	Concurrency     int     `json:"concurrency"`
	DurationSeconds int     `json:"duration_seconds"`
	Requests        int64   `json:"requests"`
	Errors          int64   `json:"errors"`
	ErrorRatePct    float64 `json:"error_rate_pct"`
	RPS             float64 `json:"rps"`
	P50Ms           float64 `json:"p50_ms"`
	P95Ms           float64 `json:"p95_ms"`
	P99Ms           float64 `json:"p99_ms"`
	RetriesDelta    uint64  `json:"retries_delta"`
	CircuitOpen     bool    `json:"circuit_open"`
	Passed          bool    `json:"passed"`
	Error           string  `json:"error,omitempty"`
}

func TestStage3StressHarness(t *testing.T) {
	if os.Getenv("STAGE3_STRESS") != "1" {
		t.Skip("set STAGE3_STRESS=1 to run stress harness")
	}

	bin := buildTestBinary(t)
	transports := []string{"streamable", "sse"}
	drivers := []string{"goburrow", "simonvetter"}
	profiles := []stressProfile{profileReadHeavy, profileMixed, profileWriteFav}
	concurrencyLevels := parseIntListEnv("STAGE3_CONCURRENCY", []int{1, 5, 10})
	duration := parseDurationEnv("STAGE3_DURATION", 3*time.Second)

	if os.Getenv("STAGE3_QUICK") == "1" {
		transports = []string{"streamable"}
		drivers = []string{"goburrow"}
		profiles = []stressProfile{profileReadHeavy}
		concurrencyLevels = []int{1}
		duration = 2 * time.Second
	}

	results := make([]stage3Result, 0)
	for _, transport := range transports {
		for _, driver := range drivers {
			for _, profile := range profiles {
				for _, conc := range concurrencyLevels {
					res := runStage3Case(t, bin, transport, driver, profile, conc, duration)
					results = append(results, res)
					if !res.Passed {
						t.Errorf("stage3 failed transport=%s driver=%s profile=%s concurrency=%d: %s", transport, driver, profile, conc, res.Error)
					}
				}
			}
		}
	}

	artifactDir := os.Getenv("STAGE3_ARTIFACT_DIR")
	if strings.TrimSpace(artifactDir) == "" {
		artifactDir = t.TempDir()
	}
	if err := writeStage3Artifacts(artifactDir, results); err != nil {
		t.Fatalf("write stage3 artifacts: %v", err)
	}
	t.Logf("stage3 artifacts written to %s", artifactDir)
}

func runStage3Case(t *testing.T, bin, transport, driver string, profile stressProfile, concurrency int, duration time.Duration) stage3Result {
	res := stage3Result{
		Transport:       transport,
		Driver:          driver,
		Profile:         string(profile),
		Concurrency:     concurrency,
		DurationSeconds: int(duration.Seconds()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration+20*time.Second)
	defer cancel()

	configPath, err := writeStage3StressConfig(t)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	cmd := exec.Command(bin, "--config", configPath, "--transport", transport, "--modbus-driver", driver)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		res.Error = fmt.Sprintf("stderr pipe: %v", err)
		return res
	}
	if err := cmd.Start(); err != nil {
		res.Error = fmt.Sprintf("start server: %v", err)
		return res
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	go io.Copy(io.Discard, stderr)

	waitForHealth(t, "http://127.0.0.1:8080/health", 5*time.Second)

	ctrl, err := connectHTTPClientSession(ctx, transport)
	if err != nil {
		res.Error = fmt.Sprintf("connect control session: %v", err)
		return res
	}
	defer ctrl.Close()

	before, _ := statusSnapshot(ctx, ctrl)

	workerStats, err := runStressWorkers(ctx, transport, profile, concurrency, duration)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	after, _ := statusSnapshot(ctx, ctrl)

	res.Requests = workerStats.requests
	res.Errors = workerStats.errors
	if res.Requests > 0 {
		res.ErrorRatePct = float64(res.Errors) / float64(res.Requests) * 100
		res.RPS = float64(res.Requests) / duration.Seconds()
	}
	res.P50Ms = percentileMs(workerStats.latencies, 50)
	res.P95Ms = percentileMs(workerStats.latencies, 95)
	res.P99Ms = percentileMs(workerStats.latencies, 99)
	if after != nil && before != nil && after.TotalRetries >= before.TotalRetries {
		res.RetriesDelta = after.TotalRetries - before.TotalRetries
		res.CircuitOpen = after.CircuitOpen
	}
	res.Passed = true
	return res
}

type stressAccum struct {
	requests  int64
	errors    int64
	latencies []time.Duration
}

func runStressWorkers(parent context.Context, transport string, profile stressProfile, concurrency int, duration time.Duration) (*stressAccum, error) {
	ctx, cancel := context.WithTimeout(parent, duration)
	defer cancel()

	var wg sync.WaitGroup
	results := make(chan stressAccum, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			acc := stressAccum{latencies: make([]time.Duration, 0, 256)}

			cs, err := connectHTTPClientSession(parent, transport)
			if err != nil {
				acc.errors++
				results <- acc
				return
			}
			defer cs.Close()

			rng := rand.New(rand.NewSource(int64(1000 + worker)))
			for {
				if ctx.Err() != nil {
					break
				}
				start := time.Now()
				err := performStressOp(ctx, cs, profile, rng.Intn(100))
				acc.requests++
				if err != nil {
					acc.errors++
				}
				acc.latencies = append(acc.latencies, time.Since(start))
			}

			results <- acc
		}(i)
	}

	wg.Wait()
	close(results)

	out := &stressAccum{latencies: make([]time.Duration, 0, concurrency*256)}
	for r := range results {
		out.requests += r.requests
		out.errors += r.errors
		out.latencies = append(out.latencies, r.latencies...)
	}
	return out, nil
}

func performStressOp(ctx context.Context, cs *mcp.ClientSession, profile stressProfile, roll int) error {
	callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var req *mcp.CallToolParams
	switch profile {
	case profileReadHeavy:
		if roll < 95 {
			req = &mcp.CallToolParams{Name: "read-input-registers", Arguments: map[string]any{"address": 0, "quantity": 1}}
		} else {
			req = &mcp.CallToolParams{Name: "get-modbus-client-status", Arguments: map[string]any{}}
		}
	case profileMixed:
		if roll < 70 {
			req = &mcp.CallToolParams{Name: "read-input-registers", Arguments: map[string]any{"address": 0, "quantity": 1}}
		} else if roll < 95 {
			req = &mcp.CallToolParams{Name: "write-holding-registers", Arguments: map[string]any{"address": 1, "values": []any{42}}}
		} else {
			req = &mcp.CallToolParams{Name: "get-modbus-client-status", Arguments: map[string]any{}}
		}
	default:
		if roll < 60 {
			req = &mcp.CallToolParams{Name: "write-tag", Arguments: map[string]any{"name": "setpoint", "holding_values": []any{42}}}
		} else {
			req = &mcp.CallToolParams{Name: "read-tag", Arguments: map[string]any{"name": "temp"}}
		}
	}

	res, err := cs.CallTool(callCtx, req)
	if err != nil {
		return err
	}
	if res.IsError {
		return fmt.Errorf("tool returned error: %s", toolResultText(res))
	}
	return nil
}

type statusView struct {
	TotalRetries uint64 `json:"total_retries"`
	CircuitOpen  bool   `json:"circuit_open"`
}

func statusSnapshot(ctx context.Context, cs *mcp.ClientSession) (*statusView, error) {
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "get-modbus-client-status", Arguments: map[string]any{}})
	if err != nil {
		return nil, err
	}
	if res.IsError {
		return nil, fmt.Errorf("status tool error: %s", toolResultText(res))
	}
	out := &statusView{}
	if err := json.Unmarshal([]byte(toolResultText(res)), out); err != nil {
		return nil, err
	}
	return out, nil
}

func connectHTTPClientSession(ctx context.Context, transport string) (*mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "stage3-client", Version: "test"}, nil)
	var tpt mcp.Transport
	switch transport {
	case "streamable":
		tpt = &mcp.StreamableClientTransport{Endpoint: "http://127.0.0.1:8080/mcp", DisableStandaloneSSE: true}
	case "sse":
		tpt = &mcp.SSEClientTransport{Endpoint: "http://127.0.0.1:8080/sse"}
	default:
		return nil, fmt.Errorf("unsupported stress transport %q", transport)
	}
	return client.Connect(ctx, tpt, nil)
}

func percentileMs(values []time.Duration, pct int) float64 {
	if len(values) == 0 {
		return 0
	}
	copyVals := append([]time.Duration(nil), values...)
	sort.Slice(copyVals, func(i, j int) bool { return copyVals[i] < copyVals[j] })
	idx := (len(copyVals) - 1) * pct / 100
	return float64(copyVals[idx].Microseconds()) / 1000.0
}

func parseIntListEnv(name string, fallback []int) []int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err == nil && v > 0 {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func parseDurationEnv(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func writeStage3StressConfig(t *testing.T) (string, error) {
	t.Helper()
	content := `mock_mode: true
write_policy:
  writes_enabled: true
  holding_write_allowlist: "0-50"
tags:
  - name: temp
    kind: holding_register
    address: 0
    quantity: 1
    access: read
    data_type: uint16
    byte_order: big
    word_order: msw
  - name: setpoint
    kind: holding_register
    address: 1
    quantity: 1
    access: read_write
    data_type: uint16
    byte_order: big
    word_order: msw
`
	path := filepath.Join(t.TempDir(), "stage3-stress.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func writeStage3Artifacts(dir string, results []stage3Result) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	jsonPath := filepath.Join(dir, "stage3-results.json")
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, b, 0o644); err != nil {
		return err
	}

	csvPath := filepath.Join(dir, "stage3-results.csv")
	f, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"transport", "driver", "profile", "concurrency", "duration_seconds", "requests", "errors", "error_rate_pct", "rps", "p50_ms", "p95_ms", "p99_ms", "retries_delta", "circuit_open", "passed", "error"}); err != nil {
		return err
	}
	for _, r := range results {
		row := []string{
			r.Transport,
			r.Driver,
			r.Profile,
			strconv.Itoa(r.Concurrency),
			strconv.Itoa(r.DurationSeconds),
			strconv.FormatInt(r.Requests, 10),
			strconv.FormatInt(r.Errors, 10),
			fmt.Sprintf("%.3f", r.ErrorRatePct),
			fmt.Sprintf("%.3f", r.RPS),
			fmt.Sprintf("%.3f", r.P50Ms),
			fmt.Sprintf("%.3f", r.P95Ms),
			fmt.Sprintf("%.3f", r.P99Ms),
			strconv.FormatUint(r.RetriesDelta, 10),
			strconv.FormatBool(r.CircuitOpen),
			strconv.FormatBool(r.Passed),
			r.Error,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return w.Error()
}
