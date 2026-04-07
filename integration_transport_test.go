package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIntegrationStdioToolsMockMode(t *testing.T) {
	bin := buildTestBinary(t)

	cmd := exec.Command(bin, "--mock-mode", "--transport", "stdio")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("failed to get stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	go io.Copy(io.Discard, stderr)
	r := bufio.NewReader(stdout)

	writeJSONLine(t, stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1"},
		},
		"id": 1,
	})
	_ = readResponseByID(t, r, 1)

	writeJSONLine(t, stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})

	writeJSONLine(t, stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "read-input-registers",
			"arguments": map[string]any{
				"address":  0,
				"quantity": 1,
			},
		},
		"id": 2,
	})
	resp2 := readResponseByID(t, r, 2)
	assertResultTextContains(t, resp2, "Input registers at address 0")

	writeJSONLine(t, stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "read-discrete-inputs",
			"arguments": map[string]any{
				"address":  0,
				"quantity": 2,
			},
		},
		"id": 3,
	})
	resp3 := readResponseByID(t, r, 3)
	assertResultTextContains(t, resp3, "Discrete inputs at address 0")

	writeJSONLine(t, stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "read-holding-registers-typed",
			"arguments": map[string]any{
				"address":   0,
				"quantity":  2,
				"data_type": "float32",
			},
		},
		"id": 4,
	})
	resp4 := readResponseByID(t, r, 4)
	assertResultTextContains(t, resp4, "Typed holding registers at address 0")
}

func TestIntegrationStreamableToolsMockMode(t *testing.T) {
	if !portAvailable("127.0.0.1:8080") {
		t.Skip("port 8080 is busy; skipping streamable integration test")
	}

	bin := buildTestBinary(t)
	cmd := exec.Command(bin, "--mock-mode", "--transport", "streamable")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("failed to get stderr pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	go io.Copy(io.Discard, stderr)

	waitForHealth(t, "http://127.0.0.1:8080/health", 5*time.Second)

	resp1 := postJSON(t, "http://127.0.0.1:8080/mcp", map[string]any{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1"},
		},
		"id": 1,
	})
	if _, ok := resp1["result"]; !ok {
		t.Fatalf("expected initialize result, got: %#v", resp1)
	}

	resp2 := postJSON(t, "http://127.0.0.1:8080/mcp", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "read-input-registers",
			"arguments": map[string]any{
				"address":  0,
				"quantity": 1,
			},
		},
		"id": 2,
	})
	assertResultTextContains(t, resp2, "Input registers at address 0")
}

func TestIntegrationStdioToolsMockModeSimonvetterConfig(t *testing.T) {
	bin := buildTestBinary(t)

	cmd := exec.Command(bin, "--mock-mode", "--modbus-driver", "simonvetter", "--transport", "stdio")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("failed to get stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	go io.Copy(io.Discard, stderr)
	r := bufio.NewReader(stdout)

	writeJSONLine(t, stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1"},
		},
		"id": 1,
	})
	_ = readResponseByID(t, r, 1)

	writeJSONLine(t, stdin, map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})

	writeJSONLine(t, stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "read-holding-registers",
			"arguments": map[string]any{
				"address":  0,
				"quantity": 1,
			},
		},
		"id": 2,
	})
	resp := readResponseByID(t, r, 2)
	assertResultTextContains(t, resp, "Holding registers at address 0")
}

func buildTestBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "modbus-server-test")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build test binary: %v, output: %s", err, string(out))
	}
	return bin
}

func writeJSONLine(t *testing.T, w io.Writer, v map[string]any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		t.Fatalf("write request: %v", err)
	}
}

func readResponseByID(t *testing.T, r *bufio.Reader, id float64) map[string]any {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_ = r
		line, err := r.ReadBytes('\n')
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var msg map[string]any
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if got, ok := msg["id"].(float64); ok && got == id {
			return msg
		}
	}
	t.Fatalf("timed out waiting for response id %.0f", id)
	return nil
}

func postJSON(t *testing.T, url string, payload map[string]any) map[string]any {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func assertResultTextContains(t *testing.T, msg map[string]any, want string) {
	t.Helper()
	result, ok := msg["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result in response: %#v", msg)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("missing content in response result: %#v", result)
	}
	entry, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("invalid content entry: %#v", content[0])
	}
	text, _ := entry["text"].(string)
	if !strings.Contains(text, want) {
		t.Fatalf("expected response text to contain %q, got %q", want, text)
	}
}

func waitForHealth(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("health endpoint did not become ready: %s", url)
}

func portAvailable(addr string) bool {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
