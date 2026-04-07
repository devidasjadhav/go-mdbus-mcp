package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type target struct {
	Name     string
	Mode     string // stdio, http-streamable, http-sse
	Cmd      []string
	Endpoint string
	Env      map[string]string
}

type result struct {
	Name        string  `json:"name"`
	Mode        string  `json:"mode"`
	Tool        string  `json:"tool"`
	Concurrency int     `json:"concurrency"`
	Requests    int64   `json:"requests"`
	Errors      int64   `json:"errors"`
	RPS         float64 `json:"rps"`
	P95MS       float64 `json:"p95_ms"`
	Status      string  `json:"status"`
	Error       string  `json:"error,omitempty"`
}

type toolPlan struct {
	tool          string
	args          map[string]any
	requiresSetup bool
}

func main() {
	base := "/tmp/mcp-servers-Uonh4R"
	modbusHost := getenvDefault("MODBUS_BENCH_HOST", "127.0.0.1")
	modbusPort := getenvDefault("MODBUS_BENCH_PORT", "5002")
	modbusPortInt := atoiDefault(modbusPort, 5002)
	targets := []target{
		{
			Name:     "go-mdbus-mcp(streamable)",
			Mode:     "http-streamable",
			Cmd:      []string{"go", "run", ".", "--transport", "streamable", "--modbus-ip", modbusHost, "--modbus-port", modbusPort, "--modbus-driver", "goburrow", "--modbus-circuit-trip-after", "9999"},
			Endpoint: "http://127.0.0.1:8080/mcp",
			Env:      map[string]string{},
		},
		{
			Name:     "go-mdbus-mcp(sse)",
			Mode:     "http-sse",
			Cmd:      []string{"go", "run", ".", "--transport", "sse", "--modbus-ip", modbusHost, "--modbus-port", modbusPort, "--modbus-driver", "goburrow", "--modbus-circuit-trip-after", "9999"},
			Endpoint: "http://127.0.0.1:8080/sse",
			Env:      map[string]string{},
		},
		{
			Name: "go-mdbus-mcp(stdio)",
			Mode: "stdio",
			Cmd:  []string{"go", "run", ".", "--transport", "stdio", "--modbus-ip", modbusHost, "--modbus-port", modbusPort, "--modbus-driver", "goburrow", "--modbus-circuit-trip-after", "9999"},
			Env:  map[string]string{},
		},
		{
			Name: "kukapay/modbus-mcp",
			Mode: "stdio",
			Cmd:  []string{"uv", "--directory", base + "/kukapay-modbus-mcp", "run", "modbus-mcp"},
			Env: map[string]string{
				"MODBUS_TYPE":             "tcp",
				"MODBUS_HOST":             modbusHost,
				"MODBUS_PORT":             modbusPort,
				"MODBUS_DEFAULT_SLAVE_ID": "1",
			},
		},
		{
			Name: "alejoseb/ModbusMCP",
			Mode: "stdio",
			Cmd:  []string{base + "/venv-alejo/bin/modbus-mcp-server", "--transport", "stdio"},
			Env:  map[string]string{},
		},
		{
			Name: "midhunxavier/MODBUS-MCP",
			Mode: "stdio",
			Cmd:  []string{"uv", "--directory", base + "/midhunxavier-MODBUS-MCP/modbus-python", "run", "modbus-mcp"},
			Env: map[string]string{
				"MODBUS_TYPE":             "tcp",
				"MODBUS_HOST":             modbusHost,
				"MODBUS_PORT":             modbusPort,
				"MODBUS_DEFAULT_SLAVE_ID": "1",
			},
		},
		{
			Name:     "ezhuk/modbus-mcp",
			Mode:     "http-streamable",
			Cmd:      []string{base + "/venv-ezhuk314/bin/modbus-mcp", "--host", "127.0.0.1", "--port", "8000"},
			Endpoint: "http://127.0.0.1:8000/mcp/",
			Env: map[string]string{
				"MODBUS_MCP_MODBUS__HOST": modbusHost,
				"MODBUS_MCP_MODBUS__PORT": modbusPort,
				"MODBUS_MCP_MODBUS__UNIT": "1",
			},
		},
		{
			Name:     "alejoseb/ModbusMCP(sse)",
			Mode:     "http-sse",
			Cmd:      []string{base + "/venv-alejo/bin/modbus-mcp-server", "--transport", "sse", "--host", "127.0.0.1", "--port", "18080"},
			Endpoint: "http://127.0.0.1:18080/sse",
			Env:      map[string]string{},
		},
	}

	if err := ensureBackend(modbusHost, modbusPortInt); err != nil {
		fmt.Printf("backend check failed: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(base + "/venv-alejo/bin/modbus-mcp-server"); err != nil {
		_ = exec.Command("python3", "-m", "venv", base+"/venv-alejo").Run()
		_ = exec.Command(base+"/venv-alejo/bin/pip", "install", "-q", "-e", base+"/alejoseb-ModbusMCP").Run()
	}

	all := make([]result, 0)
	for _, t := range targets {
		fmt.Printf("\n== Testing %s ==\n", t.Name)
		rows := runTarget(t)
		all = append(all, rows...)
	}

	out, _ := json.MarshalIndent(all, "", "  ")
	_ = os.WriteFile("/tmp/mcp-servers-Uonh4R/compare-results.json", out, 0o644)

	fmt.Println("\n| Server | Concurrency | Tool | RPS | p95 ms | Error % | Status |")
	fmt.Println("|---|---:|---|---:|---:|---:|---|")
	for _, r := range all {
		errPct := 0.0
		if r.Requests > 0 {
			errPct = float64(r.Errors) / float64(r.Requests) * 100
		}
		fmt.Printf("| %s | %d | %s | %.1f | %.2f | %.2f | %s |\n", r.Name, r.Concurrency, r.Tool, r.RPS, r.P95MS, errPct, r.Status)
	}
	fmt.Println("\nSaved JSON: /tmp/mcp-servers-Uonh4R/compare-results.json")
}

func ensureBackend(host string, port int) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	if conn, err := net.DialTimeout("tcp", addr, 800*time.Millisecond); err == nil {
		_ = conn.Close()
		return nil
	}
	return fmt.Errorf("cannot connect to modbus backend %s", addr)
}

func runTarget(t target) []result {
	res := make([]result, 0, 2)
	for _, conc := range []int{1, 5} {
		r := result{Name: t.Name, Mode: t.Mode, Concurrency: conc, Status: "ok"}
		if err := runCase(t, conc, &r); err != nil {
			r.Status = "failed"
			r.Error = err.Error()
		}
		res = append(res, r)
	}
	return res
}

func runCase(t target, conc int, out *result) error {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	var proc *exec.Cmd
	if strings.HasPrefix(t.Mode, "http") {
		if err := waitEndpointClosed(t.Endpoint, 5*time.Second); err != nil {
			return err
		}
		proc = exec.Command(t.Cmd[0], t.Cmd[1:]...)
		proc.Env = append(os.Environ(), flattenEnv(t.Env)...)
		if err := proc.Start(); err != nil {
			return fmt.Errorf("start http server: %w", err)
		}
		defer func() {
			_ = proc.Process.Kill()
			_ = proc.Wait()
		}()
		if err := waitHTTP(t.Endpoint, 10*time.Second); err != nil {
			return err
		}
	}

	listSession, list, err := connectAndList(ctx, t)
	if err != nil {
		return err
	}

	plan, err := pickWorkingTool(ctx, listSession, list)
	_ = listSession.Close()
	if err != nil {
		return err
	}
	out.Tool = plan.tool

	start := time.Now()
	deadline := start.Add(8 * time.Second)
	var wg sync.WaitGroup
	var mu sync.Mutex
	lats := make([]time.Duration, 0, 1024)
	for i := 0; i < conc; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			connCtx, connCancel := context.WithTimeout(context.Background(), 12*time.Second)
			cs, _, err := connectAndList(connCtx, t)
			connCancel()
			if err != nil {
				mu.Lock()
				out.Errors++
				out.Requests++
				mu.Unlock()
				return
			}
			defer cs.Close()

			sessionArgs, err := prepareSessionArgs(context.Background(), cs, plan)
			if err != nil {
				mu.Lock()
				out.Errors++
				out.Requests++
				if out.Error == "" {
					out.Error = err.Error()
				}
				mu.Unlock()
				return
			}

			for time.Now().Before(deadline) {
				rctx, rcancel := context.WithTimeout(context.Background(), 3*time.Second)
				ts := time.Now()
				res, err := cs.CallTool(rctx, &mcp.CallToolParams{Name: plan.tool, Arguments: sessionArgs})
				rcancel()
				mu.Lock()
				out.Requests++
				lats = append(lats, time.Since(ts))
				if err != nil || res == nil || res.IsError {
					out.Errors++
					if out.Error == "" {
						if err != nil {
							out.Error = err.Error()
						} else if res != nil {
							out.Error = toolResultText(res)
						}
					}
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	elapsed := time.Since(start).Seconds()
	if elapsed > 0 {
		out.RPS = float64(out.Requests) / elapsed
	}
	out.P95MS = p95(lats)
	if out.Requests == 0 {
		out.Status = "failed"
		return fmt.Errorf("no successful requests")
	}
	if out.Errors == out.Requests {
		out.Status = "failed"
		if out.Error != "" {
			return fmt.Errorf("all requests failed: %s", out.Error)
		}
		return fmt.Errorf("all requests failed")
	}
	return nil
}

func connectAndList(ctx context.Context, t target) (*mcp.ClientSession, *mcp.ListToolsResult, error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "compare", Version: "1"}, nil)
	var (
		cs  *mcp.ClientSession
		err error
	)
	if t.Mode == "stdio" {
		cmd := exec.Command(t.Cmd[0], t.Cmd[1:]...)
		cmd.Env = append(os.Environ(), flattenEnv(t.Env)...)
		cs, err = client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("stdio connect: %w", err)
		}
	} else if t.Mode == "http-streamable" {
		cs, err = client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: t.Endpoint, DisableStandaloneSSE: true}, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("http connect: %w", err)
		}
	} else if t.Mode == "http-sse" {
		cs, err = client.Connect(ctx, &mcp.SSEClientTransport{Endpoint: t.Endpoint}, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("sse connect: %w", err)
		}
	} else {
		return nil, nil, fmt.Errorf("unsupported mode %q", t.Mode)
	}
	list, err := cs.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		_ = cs.Close()
		return nil, nil, fmt.Errorf("list tools: %w", err)
	}
	return cs, list, nil
}

func pickTool(list *mcp.ListToolsResult) (string, map[string]any) {
	if list == nil {
		return "", nil
	}
	available := map[string]bool{}
	for _, t := range list.Tools {
		available[t.Name] = true
	}
	candidates := []struct {
		name string
		args map[string]any
	}{
		{"read-input-registers", map[string]any{"address": 0, "quantity": 1}},
		{"read_input_registers", map[string]any{"address": 0, "count": 1, "slave_id": 1}},
		{"read_register", map[string]any{"address": 0, "slave_id": 1}},
		{"read_registers", map[string]any{"address": 40001, "count": 1}},
	}
	for _, c := range candidates {
		if available[c.name] {
			return c.name, c.args
		}
	}
	return "", nil
}

func pickWorkingTool(ctx context.Context, cs *mcp.ClientSession, list *mcp.ListToolsResult) (toolPlan, error) {
	if list == nil {
		return toolPlan{}, fmt.Errorf("no tools listed")
	}

	candidates := []toolPlan{
		{tool: "read-input-registers", args: map[string]any{"address": 0, "quantity": 1}},
		{tool: "read-holding-registers", args: map[string]any{"address": 0, "quantity": 1}},
		{tool: "read_input_registers", args: map[string]any{"address": 0, "count": 1, "slave_id": 1}},
		{tool: "read_holding_registers", args: map[string]any{"address": 0, "count": 1, "slave_id": 1}},
		{tool: "read_input_registers", args: map[string]any{"address": 0, "count": 1, "client_id": "__SESSION_CLIENT_ID__"}, requiresSetup: true},
		{tool: "read_holding_registers", args: map[string]any{"address": 0, "count": 1, "client_id": "__SESSION_CLIENT_ID__"}, requiresSetup: true},
		{tool: "read_register", args: map[string]any{"address": 0, "slave_id": 1}},
		{tool: "read_registers", args: map[string]any{"address": 40001, "count": 1}},
		{tool: "read_registers", args: map[string]any{"address": 0, "count": 1}},
		{tool: "read_registers", args: map[string]any{"host": getenvDefault("MODBUS_BENCH_HOST", "127.0.0.1"), "port": atoiDefault(getenvDefault("MODBUS_BENCH_PORT", "5002"), 5002), "address": 40001, "count": 1, "unit": 1}},
		{tool: "read_registers", args: map[string]any{"host": getenvDefault("MODBUS_BENCH_HOST", "127.0.0.1"), "port": atoiDefault(getenvDefault("MODBUS_BENCH_PORT", "5002"), 5002), "address": 0, "count": 1, "unit": 1}},
	}

	for _, cand := range candidates {
		if !hasTool(list, cand.tool) {
			continue
		}
		args, err := prepareSessionArgs(ctx, cs, cand)
		if err != nil {
			continue
		}
		rctx, cancel := context.WithTimeout(ctx, 4*time.Second)
		res, err := cs.CallTool(rctx, &mcp.CallToolParams{Name: cand.tool, Arguments: args})
		cancel()
		if err == nil && res != nil && !res.IsError {
			return cand, nil
		}
	}

	tool, args := pickTool(list)
	if tool == "" {
		return toolPlan{}, fmt.Errorf("no known read tool found")
	}
	return toolPlan{tool: tool, args: args}, nil
}

func prepareSessionArgs(ctx context.Context, cs *mcp.ClientSession, plan toolPlan) (map[string]any, error) {
	args := cloneMap(plan.args)
	if !plan.requiresSetup {
		return args, nil
	}

	rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	res, err := cs.CallTool(rctx, &mcp.CallToolParams{
		Name: "create_tcp_client",
		Arguments: map[string]any{
			"host":     getenvDefault("MODBUS_BENCH_HOST", "127.0.0.1"),
			"port":     atoiDefault(getenvDefault("MODBUS_BENCH_PORT", "5002"), 5002),
			"slave_id": 1,
		},
	})
	if err != nil {
		return nil, err
	}
	if res == nil || res.IsError {
		return nil, fmt.Errorf("create_tcp_client failed: %s", toolResultText(res))
	}
	cid := extractClientID(res)
	if cid == "" {
		return nil, fmt.Errorf("create_tcp_client returned no client_id")
	}
	for k, v := range args {
		if s, ok := v.(string); ok && s == "__SESSION_CLIENT_ID__" {
			args[k] = cid
		}
	}
	return args, nil
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func hasTool(list *mcp.ListToolsResult, name string) bool {
	if list == nil {
		return false
	}
	for _, t := range list.Tools {
		if t.Name == name {
			return true
		}
	}
	return false
}

func extractClientID(res *mcp.CallToolResult) string {
	txt := toolResultText(res)
	if strings.TrimSpace(txt) == "" {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(txt), &obj); err == nil {
		if cid, ok := obj["client_id"].(string); ok && cid != "" {
			return cid
		}
		if data, ok := obj["data"].(map[string]any); ok {
			if cid, ok := data["client_id"].(string); ok && cid != "" {
				return cid
			}
		}
	}
	re := regexp.MustCompile(`client_id['"\s:=]+([a-zA-Z0-9._-]+)`)
	m := re.FindStringSubmatch(txt)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

func toolResultText(res *mcp.CallToolResult) string {
	if res == nil {
		return ""
	}
	parts := make([]string, 0, len(res.Content))
	for _, c := range res.Content {
		if v, ok := c.(*mcp.TextContent); ok {
			parts = append(parts, v.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func p95(values []time.Duration) float64 {
	if len(values) == 0 {
		return 0
	}
	copyVals := append([]time.Duration(nil), values...)
	sort.Slice(copyVals, func(i, j int) bool { return copyVals[i] < copyVals[j] })
	idx := (len(copyVals) - 1) * 95 / 100
	return float64(copyVals[idx].Microseconds()) / 1000.0
}

func flattenEnv(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}

func waitHTTP(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("endpoint not ready: %s", url)
}

func waitEndpointClosed(raw string, timeout time.Duration) error {
	u, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	host := u.Hostname()
	port := u.Port()
	if host == "" || port == "" {
		return nil
	}
	addr := net.JoinHostPort(host, port)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err != nil {
			return nil
		}
		_ = c.Close()
		time.Sleep(120 * time.Millisecond)
	}
	return fmt.Errorf("endpoint still busy at %s", addr)
}

func getenvDefault(name, fallback string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	return v
}

func atoiDefault(s string, fallback int) int {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
