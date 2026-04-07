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

type stage1Result struct {
	Transport string `json:"transport"`
	Driver    string `json:"driver"`
	Passed    bool   `json:"passed"`
	Duration  string `json:"duration"`
	Error     string `json:"error,omitempty"`
}

func TestStage1MatrixMockMode(t *testing.T) {
	bin := buildTestBinary(t)

	transports := []string{"stdio", "streamable", "sse"}
	drivers := []string{"goburrow", "simonvetter"}
	results := make([]stage1Result, 0, len(transports)*len(drivers))

	for _, transport := range transports {
		for _, driver := range drivers {
			start := time.Now()
			err := runStage1Case(t, bin, transport, driver)
			res := stage1Result{
				Transport: transport,
				Driver:    driver,
				Passed:    err == nil,
				Duration:  time.Since(start).String(),
			}
			if err != nil {
				res.Error = err.Error()
				t.Errorf("stage1 case failed transport=%s driver=%s: %v", transport, driver, err)
			}
			results = append(results, res)
		}
	}

	artifactDir := os.Getenv("STAGE1_ARTIFACT_DIR")
	if strings.TrimSpace(artifactDir) == "" {
		artifactDir = t.TempDir()
	}
	if err := writeStage1Artifacts(artifactDir, results); err != nil {
		t.Fatalf("write stage1 artifacts: %v", err)
	}
	t.Logf("stage1 artifacts written to %s", artifactDir)
}

func runStage1Case(t *testing.T, bin, transport, driver string) error {
	t.Helper()

	if transport != "stdio" {
		if !portAvailable("127.0.0.1:8080") {
			return fmt.Errorf("port 8080 is busy")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var cs *mcp.ClientSession
	var cleanup func()

	switch transport {
	case "stdio":
		client := mcp.NewClient(&mcp.Implementation{Name: "stage1-client", Version: "test"}, nil)
		cmd := exec.Command(bin, "--mock-mode", "--transport", transport, "--modbus-driver", driver)
		tpt := &mcp.CommandTransport{Command: cmd}
		session, err := client.Connect(ctx, tpt, nil)
		if err != nil {
			return fmt.Errorf("connect stdio: %w", err)
		}
		cs = session
		cleanup = func() { _ = cs.Close() }
	case "streamable", "sse":
		cmd := exec.Command(bin, "--mock-mode", "--transport", transport, "--modbus-driver", driver)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("stderr pipe: %w", err)
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start server: %w", err)
		}
		go io.Copy(io.Discard, stderr)

		waitForHealth(t, "http://127.0.0.1:8080/health", 5*time.Second)

		client := mcp.NewClient(&mcp.Implementation{Name: "stage1-client", Version: "test"}, nil)
		var tpt mcp.Transport
		if transport == "streamable" {
			tpt = &mcp.StreamableClientTransport{Endpoint: "http://127.0.0.1:8080/mcp", DisableStandaloneSSE: true}
		} else {
			tpt = &mcp.SSEClientTransport{Endpoint: "http://127.0.0.1:8080/sse"}
		}
		session, err := client.Connect(ctx, tpt, nil)
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return fmt.Errorf("connect %s: %w", transport, err)
		}
		cs = session
		cleanup = func() {
			_ = cs.Close()
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	default:
		return fmt.Errorf("unsupported transport %q", transport)
	}
	defer cleanup()

	if err := runStage1Assertions(ctx, cs); err != nil {
		return err
	}

	return nil
}

func runStage1Assertions(ctx context.Context, cs *mcp.ClientSession) error {
	tools, err := cs.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}
	if len(tools.Tools) == 0 {
		return fmt.Errorf("list tools returned zero tools")
	}
	if !hasToolName(tools, "get-modbus-client-status") {
		return fmt.Errorf("missing get-modbus-client-status tool")
	}

	status, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "get-modbus-client-status", Arguments: map[string]any{}})
	if err != nil {
		return fmt.Errorf("call status tool: %w", err)
	}
	if status.IsError {
		return fmt.Errorf("status tool returned error: %s", toolResultText(status))
	}

	read, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "read-input-registers",
		Arguments: map[string]any{
			"address":  0,
			"quantity": 1,
		},
	})
	if err != nil {
		return fmt.Errorf("call read-input-registers: %w", err)
	}
	if read.IsError || !strings.Contains(toolResultText(read), "Input registers at address 0") {
		return fmt.Errorf("unexpected read-input-registers response: %s", toolResultText(read))
	}

	bad, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "read-input-registers",
		Arguments: map[string]any{
			"address":  0,
			"quantity": 0,
		},
	})
	if err != nil {
		return fmt.Errorf("call invalid read-input-registers: %w", err)
	}
	if !bad.IsError {
		return fmt.Errorf("expected invalid read to return IsError=true")
	}

	return nil
}

func hasToolName(tools *mcp.ListToolsResult, name string) bool {
	if tools == nil {
		return false
	}
	for _, t := range tools.Tools {
		if t.Name == name {
			return true
		}
	}
	return false
}

func toolResultText(res *mcp.CallToolResult) string {
	if res == nil {
		return ""
	}
	parts := make([]string, 0, len(res.Content))
	for _, c := range res.Content {
		switch v := c.(type) {
		case *mcp.TextContent:
			parts = append(parts, v.Text)
		default:
			parts = append(parts, fmt.Sprintf("%T", c))
		}
	}
	return strings.Join(parts, "\n")
}

func writeStage1Artifacts(dir string, results []stage1Result) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	jsonPath := filepath.Join(dir, "stage1-results.json")
	jsonBytes, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, jsonBytes, 0o644); err != nil {
		return err
	}

	csvPath := filepath.Join(dir, "stage1-results.csv")
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
