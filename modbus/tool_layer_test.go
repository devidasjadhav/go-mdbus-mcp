package modbus

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeDriver struct {
	status      ClientStatus
	readHolding []byte
	err         error
}

func (f *fakeDriver) DriverName() string { return "fake" }
func (f *fakeDriver) TransportMode() string {
	return "tcp"
}
func (f *fakeDriver) Execute(ctx context.Context, slaveID uint8, allowRetry bool, operation func() (*mcp.CallToolResult, error)) (*mcp.CallToolResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return operation()
}
func (f *fakeDriver) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.readHolding, nil
}
func (f *fakeDriver) ReadInputRegisters(address, quantity uint16) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []byte{0x00, 0x01}, nil
}
func (f *fakeDriver) ReadCoils(address, quantity uint16) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []byte{0x01}, nil
}
func (f *fakeDriver) ReadDiscreteInputs(address, quantity uint16) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []byte{0x00}, nil
}
func (f *fakeDriver) WriteSingleRegister(address, value uint16) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []byte{0x00, 0x00, 0x00, 0x00}, nil
}
func (f *fakeDriver) WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []byte{0x00, 0x00, 0x00, 0x00}, nil
}
func (f *fakeDriver) WriteMultipleCoils(address, quantity uint16, value []byte) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []byte{0x00, 0x00, 0x00, 0x00}, nil
}
func (f *fakeDriver) Status() ClientStatus { return f.status }
func (f *fakeDriver) Close() error         { return nil }

func TestToolLayerStatusAndListTools(t *testing.T) {
	driver := &fakeDriver{status: ClientStatus{Driver: "fake", Mode: "tcp", TotalOperations: 7}}
	cs, cleanup := newToolTestClientSession(t, driver, &WritePolicy{enabled: false}, nil)
	defer cleanup()

	tools, err := cs.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}
	if len(tools.Tools) < 5 {
		t.Fatalf("expected several registered tools, got %d", len(tools.Tools))
	}

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "get-modbus-client-status", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("status tool call failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected non-error status result")
	}
	text := toolResultText(res)
	if !strings.Contains(text, "\"driver\": \"fake\"") {
		t.Fatalf("unexpected status payload: %s", text)
	}
}

func TestToolLayerReadHoldingRejectsOddBytes(t *testing.T) {
	driver := &fakeDriver{readHolding: []byte{0x00, 0x2A, 0xFF}}
	cs, cleanup := newToolTestClientSession(t, driver, &WritePolicy{enabled: true}, nil)
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "read-holding-registers",
		Arguments: map[string]any{
			"address":  0,
			"quantity": 1,
		},
	})
	if err != nil {
		t.Fatalf("tool call failed: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected error result for odd response payload")
	}
	if !strings.Contains(toolResultText(res), "invalid holding register response") {
		t.Fatalf("unexpected error message: %s", toolResultText(res))
	}
}

func TestToolLayerWriteHoldingBlockedByPolicy(t *testing.T) {
	driver := &fakeDriver{}
	cs, cleanup := newToolTestClientSession(t, driver, &WritePolicy{enabled: false}, nil)
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "write-holding-registers",
		Arguments: map[string]any{
			"address": 0,
			"values":  []any{1},
		},
	})
	if err != nil {
		t.Fatalf("tool call failed: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected policy-blocked write to return IsError=true")
	}
	if !strings.Contains(toolResultText(res), "guarded rejection") {
		t.Fatalf("unexpected write-policy error: %s", toolResultText(res))
	}
}

func TestToolLayerWriteTagAmbiguousInput(t *testing.T) {
	tagMap, err := NewTagMap([]TagDef{{Name: "setpoint", Kind: TagKindHolding, Address: 10, Quantity: 1, Access: TagAccessReadWrite, DataType: "uint16", ByteOrder: "big", WordOrder: "msw", Scale: 1, ScaleSet: true}})
	if err != nil {
		t.Fatalf("failed to build tag map: %v", err)
	}

	driver := &fakeDriver{}
	cs, cleanup := newToolTestClientSession(t, driver, &WritePolicy{enabled: true}, tagMap)
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "write-tag",
		Arguments: map[string]any{
			"name":           "setpoint",
			"holding_values": []any{5},
			"numeric_value":  5,
		},
	})
	if err != nil {
		t.Fatalf("tool call failed: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected ambiguous input to return IsError=true")
	}
	if !strings.Contains(toolResultText(res), "ambiguous input") {
		t.Fatalf("unexpected error message: %s", toolResultText(res))
	}
}

func TestToolLayerReadTagHoldingSuccess(t *testing.T) {
	tagMap, err := NewTagMap([]TagDef{{Name: "temp", Kind: TagKindHolding, Address: 1, Quantity: 1, Access: TagAccessRead, DataType: "uint16", ByteOrder: "big", WordOrder: "msw", Scale: 1, ScaleSet: true}})
	if err != nil {
		t.Fatalf("failed to build tag map: %v", err)
	}

	driver := &fakeDriver{readHolding: []byte{0x00, 0x2A}}
	cs, cleanup := newToolTestClientSession(t, driver, &WritePolicy{enabled: true}, tagMap)
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "read-tag",
		Arguments: map[string]any{
			"name": "temp",
		},
	})
	if err != nil {
		t.Fatalf("tool call failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected read-tag success, got error: %s", toolResultText(res))
	}
	text := toolResultText(res)
	if !strings.Contains(text, "\"decoded_value\": 42") {
		t.Fatalf("unexpected read-tag payload: %s", text)
	}
}

func newToolTestClientSession(t *testing.T, driver Driver, writePolicy *WritePolicy, tagMap *TagMap) (*mcp.ClientSession, func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	server := mcp.NewServer(&mcp.Implementation{Name: "tool-test-server", Version: "test"}, nil)
	RegisterTools(server, driver, writePolicy, tagMap)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "tool-test-client", Version: "test"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		cancel()
		t.Fatalf("client connect failed: %v", err)
	}

	cleanup := func() {
		_ = cs.Close()
		cancel()
	}

	return cs, cleanup
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
