package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type stage2Result struct {
	Transport string `json:"transport"`
	Driver    string `json:"driver"`
	Passed    bool   `json:"passed"`
	Duration  string `json:"duration"`
	Error     string `json:"error,omitempty"`
}

func TestStage2AdvancedMatrixMockMode(t *testing.T) {
	bin := buildTestBinary(t)

	transports := []string{"stdio", "streamable", "sse"}
	drivers := []string{"goburrow", "simonvetter"}
	results := make([]stage2Result, 0, len(transports)*len(drivers))

	for _, transport := range transports {
		for _, driver := range drivers {
			start := time.Now()
			err := runStage2Case(t, bin, transport, driver)
			res := stage2Result{
				Transport: transport,
				Driver:    driver,
				Passed:    err == nil,
				Duration:  time.Since(start).String(),
			}
			if err != nil {
				res.Error = err.Error()
				t.Errorf("stage2 case failed transport=%s driver=%s: %v", transport, driver, err)
			}
			results = append(results, res)
		}
	}

	artifactDir := os.Getenv("STAGE2_ARTIFACT_DIR")
	if strings.TrimSpace(artifactDir) == "" {
		artifactDir = t.TempDir()
	}
	if err := writeStage2Artifacts(artifactDir, results); err != nil {
		t.Fatalf("write stage2 artifacts: %v", err)
	}
	t.Logf("stage2 artifacts written to %s", artifactDir)
}

func runStage2Case(t *testing.T, bin, transport, driver string) error {
	t.Helper()

	if transport != "stdio" && !portAvailable("127.0.0.1:8080") {
		return fmt.Errorf("port 8080 is busy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	enabledCfg, err := writeStage2Config(t, true)
	if err != nil {
		return err
	}

	cs, cleanup, err := connectStage2Client(t, ctx, bin, transport, driver, enabledCfg)
	if err != nil {
		return err
	}

	if err := assertStage2EnabledLane(ctx, cs); err != nil {
		cleanup()
		return err
	}
	cleanup()

	disabledCfg, err := writeStage2Config(t, false)
	if err != nil {
		return err
	}

	cs2, cleanup2, err := connectStage2Client(t, ctx, bin, transport, driver, disabledCfg)
	if err != nil {
		return err
	}
	defer cleanup2()

	if err := assertStage2DisabledLane(ctx, cs2); err != nil {
		return err
	}

	return nil
}

func connectStage2Client(t *testing.T, ctx context.Context, bin, transport, driver, configPath string) (*mcp.ClientSession, func(), error) {
	t.Helper()

	args := []string{"--config", configPath, "--transport", transport, "--modbus-driver", driver}
	client := mcp.NewClient(&mcp.Implementation{Name: "stage2-client", Version: "test"}, nil)

	switch transport {
	case "stdio":
		cmd := exec.Command(bin, args...)
		tpt := &mcp.CommandTransport{Command: cmd}
		cs, err := client.Connect(ctx, tpt, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("connect stdio: %w", err)
		}
		return cs, func() { _ = cs.Close() }, nil

	case "streamable", "sse":
		cmd := exec.Command(bin, args...)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return nil, nil, fmt.Errorf("stderr pipe: %w", err)
		}
		if err := cmd.Start(); err != nil {
			return nil, nil, fmt.Errorf("start server: %w", err)
		}
		go io.Copy(io.Discard, stderr)
		waitForHealth(t, "http://127.0.0.1:8080/health", 5*time.Second)

		var tpt mcp.Transport
		if transport == "streamable" {
			tpt = &mcp.StreamableClientTransport{Endpoint: "http://127.0.0.1:8080/mcp", DisableStandaloneSSE: true}
		} else {
			tpt = &mcp.SSEClientTransport{Endpoint: "http://127.0.0.1:8080/sse"}
		}
		cs, err := client.Connect(ctx, tpt, nil)
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return nil, nil, fmt.Errorf("connect %s: %w", transport, err)
		}
		return cs, func() {
			_ = cs.Close()
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}, nil
	default:
		return nil, nil, fmt.Errorf("unsupported transport %q", transport)
	}
}

func assertStage2EnabledLane(ctx context.Context, cs *mcp.ClientSession) error {
	list, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "list-tags", Arguments: map[string]any{}})
	if err != nil {
		return fmt.Errorf("list-tags call: %w", err)
	}
	if list.IsError || !strings.Contains(toolResultText(list), "setpoint") {
		return fmt.Errorf("unexpected list-tags response: %s", toolResultText(list))
	}

	read, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "read-tag", Arguments: map[string]any{"name": "temp"}})
	if err != nil {
		return fmt.Errorf("read-tag call: %w", err)
	}
	if read.IsError {
		return fmt.Errorf("read-tag returned error: %s", toolResultText(read))
	}

	writeOK, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "write-tag", Arguments: map[string]any{"name": "setpoint", "holding_values": []any{55}}})
	if err != nil {
		return fmt.Errorf("write-tag call: %w", err)
	}
	if writeOK.IsError {
		return fmt.Errorf("write-tag expected success, got: %s", toolResultText(writeOK))
	}

	ambiguous, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "write-tag", Arguments: map[string]any{"name": "setpoint", "holding_values": []any{5}, "numeric_value": 5}})
	if err != nil {
		return fmt.Errorf("write-tag ambiguous call: %w", err)
	}
	if !ambiguous.IsError || !strings.Contains(toolResultText(ambiguous), "ambiguous input") {
		return fmt.Errorf("expected ambiguous-input rejection, got: %s", toolResultText(ambiguous))
	}

	blocked, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "write-holding-registers", Arguments: map[string]any{"address": 200, "values": []any{1}}})
	if err != nil {
		return fmt.Errorf("write-holding-registers blocked call: %w", err)
	}
	if !blocked.IsError {
		return fmt.Errorf("expected out-of-allowlist write to fail")
	}

	return nil
}

func assertStage2DisabledLane(ctx context.Context, cs *mcp.ClientSession) error {
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "write-holding-registers", Arguments: map[string]any{"address": 1, "values": []any{1}}})
	if err != nil {
		return fmt.Errorf("disabled write call: %w", err)
	}
	if !res.IsError || !strings.Contains(toolResultText(res), "guarded rejection") {
		return fmt.Errorf("expected guarded rejection when writes disabled, got: %s", toolResultText(res))
	}
	return nil
}

func writeStage2Config(t *testing.T, writesEnabled bool) (string, error) {
	t.Helper()

	tpl := `mock_mode: true
transport: streamable
write_policy:
  writes_enabled: %t
  holding_write_allowlist: "0-20"
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
	content := fmt.Sprintf(tpl, writesEnabled)
	path := filepath.Join(t.TempDir(), fmt.Sprintf("stage2-%t.yaml", writesEnabled))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write stage2 config: %w", err)
	}
	return path, nil
}

func writeStage2Artifacts(dir string, results []stage2Result) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	jsonPath := filepath.Join(dir, "stage2-results.json")
	jsonBytes, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, jsonBytes, 0o644); err != nil {
		return err
	}

	csvPath := filepath.Join(dir, "stage2-results.csv")
	f, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"transport", "driver", "passed", "duration", "error"}); err != nil {
		return err
	}
	for _, r := range results {
		if err := w.Write([]string{r.Transport, r.Driver, fmt.Sprintf("%t", r.Passed), r.Duration, r.Error}); err != nil {
			return err
		}
	}
	return w.Error()
}
