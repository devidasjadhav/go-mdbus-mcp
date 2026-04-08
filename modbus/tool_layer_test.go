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
	lastSingle  struct {
		address uint16
		value   uint16
	}
	lastMulti struct {
		address  uint16
		quantity uint16
		value    []byte
	}
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
	f.lastSingle.address = address
	f.lastSingle.value = value
	return []byte{0x00, 0x00, 0x00, 0x00}, nil
}
func (f *fakeDriver) WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastMulti.address = address
	f.lastMulti.quantity = quantity
	f.lastMulti.value = append([]byte(nil), value...)
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

func TestToolLayerWriteHoldingTypedFloat32Success(t *testing.T) {
	driver := &fakeDriver{}
	cs, cleanup := newToolTestClientSession(t, driver, &WritePolicy{enabled: true}, nil)
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "write-holding-registers-typed",
		Arguments: map[string]any{
			"address":       7,
			"data_type":     "float32",
			"numeric_value": 12.5,
		},
	})
	if err != nil {
		t.Fatalf("tool call failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", toolResultText(res))
	}
	if driver.lastMulti.address != 7 {
		t.Fatalf("unexpected write address: %d", driver.lastMulti.address)
	}
	if driver.lastMulti.quantity != 2 {
		t.Fatalf("unexpected write quantity: %d", driver.lastMulti.quantity)
	}
	if got := wordsFromBytes(driver.lastMulti.value); len(got) != 2 || got[0] != 0x4148 || got[1] != 0x0000 {
		t.Fatalf("unexpected encoded float32 words: %v", got)
	}
}

func TestToolLayerWriteHoldingTypedStringSuccess(t *testing.T) {
	driver := &fakeDriver{}
	cs, cleanup := newToolTestClientSession(t, driver, &WritePolicy{enabled: true}, nil)
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "write-holding-registers-typed",
		Arguments: map[string]any{
			"address":      9,
			"data_type":    "string",
			"quantity":     2,
			"string_value": "ABC",
		},
	})
	if err != nil {
		t.Fatalf("tool call failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", toolResultText(res))
	}
	if driver.lastMulti.address != 9 {
		t.Fatalf("unexpected write address: %d", driver.lastMulti.address)
	}
	if driver.lastMulti.quantity != 2 {
		t.Fatalf("unexpected write quantity: %d", driver.lastMulti.quantity)
	}
	if got := wordsFromBytes(driver.lastMulti.value); len(got) != 2 || got[0] != 0x4142 || got[1] != 0x4300 {
		t.Fatalf("unexpected encoded string words: %v", got)
	}
}

func TestToolLayerWriteHoldingTypedRejectsAmbiguousInput(t *testing.T) {
	driver := &fakeDriver{}
	cs, cleanup := newToolTestClientSession(t, driver, &WritePolicy{enabled: true}, nil)
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "write-holding-registers-typed",
		Arguments: map[string]any{
			"address":       0,
			"data_type":     "uint16",
			"numeric_value": 1,
			"string_value":  "1",
		},
	})
	if err != nil {
		t.Fatalf("tool call failed: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected ambiguous input to return IsError=true")
	}
	if !strings.Contains(toolResultText(res), "provide exactly one") {
		t.Fatalf("unexpected error message: %s", toolResultText(res))
	}
}

func TestToolLayerWriteHoldingTypedThenReadHoldingRegisters(t *testing.T) {
	driver, err := NewDriver(&Config{UseMock: true, MockRegisters: 64, MockCoils: 64})
	if err != nil {
		t.Fatalf("failed to create mock driver: %v", err)
	}
	defer func() { _ = driver.Close() }()

	cs, cleanup := newToolTestClientSession(t, driver, &WritePolicy{enabled: true}, nil)
	defer cleanup()

	tests := []struct {
		name    string
		address uint16
		args    map[string]any
		qty     uint16
		want    []uint16
	}{
		{
			name:    "uint16",
			address: 0,
			args: map[string]any{
				"address":       0,
				"data_type":     "uint16",
				"numeric_value": 65535,
			},
			qty:  1,
			want: []uint16{0xFFFF},
		},
		{
			name:    "int16 negative",
			address: 1,
			args: map[string]any{
				"address":       1,
				"data_type":     "int16",
				"numeric_value": -2,
			},
			qty:  1,
			want: []uint16{0xFFFE},
		},
		{
			name:    "uint32",
			address: 2,
			args: map[string]any{
				"address":       2,
				"data_type":     "uint32",
				"numeric_value": 65537,
			},
			qty:  2,
			want: []uint16{0x0001, 0x0001},
		},
		{
			name:    "int32 negative",
			address: 4,
			args: map[string]any{
				"address":       4,
				"data_type":     "int32",
				"numeric_value": -42,
			},
			qty:  2,
			want: []uint16{0xFFFF, 0xFFD6},
		},
		{
			name:    "float32",
			address: 6,
			args: map[string]any{
				"address":       6,
				"data_type":     "float32",
				"numeric_value": 12.5,
			},
			qty:  2,
			want: []uint16{0x4148, 0x0000},
		},
		{
			name:    "float32 byte and word order",
			address: 8,
			args: map[string]any{
				"address":       8,
				"data_type":     "float32",
				"numeric_value": 12.5,
				"byte_order":    "little",
				"word_order":    "lsw",
			},
			qty:  2,
			want: []uint16{0x0000, 0x4841},
		},
		{
			name:    "string",
			address: 10,
			args: map[string]any{
				"address":      10,
				"data_type":    "string",
				"quantity":     2,
				"string_value": "ABC",
			},
			qty:  2,
			want: []uint16{0x4142, 0x4300},
		},
		{
			name:    "scale and offset",
			address: 12,
			args: map[string]any{
				"address":       12,
				"data_type":     "uint16",
				"numeric_value": 52,
				"scale":         2,
				"offset":        10,
			},
			qty:  1,
			want: []uint16{21},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			writeRes, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "write-holding-registers-typed", Arguments: tc.args})
			if err != nil {
				t.Fatalf("typed write failed: %v", err)
			}
			if writeRes.IsError {
				t.Fatalf("typed write returned error: %s", toolResultText(writeRes))
			}

			readRes, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
				Name: "read-holding-registers",
				Arguments: map[string]any{
					"address":  tc.address,
					"quantity": tc.qty,
				},
			})
			if err != nil {
				t.Fatalf("read holding registers failed: %v", err)
			}
			if readRes.IsError {
				t.Fatalf("read holding registers returned error: %s", toolResultText(readRes))
			}
			wantText := fmt.Sprintf("Holding registers at address %d: %v", tc.address, tc.want)
			if !strings.Contains(toolResultText(readRes), wantText) {
				t.Fatalf("unexpected holding-register read. want %q, got %q", wantText, toolResultText(readRes))
			}
		})
	}
}

func TestToolLayerWriteHoldingTypedCornerCases(t *testing.T) {
	driver, err := NewDriver(&Config{UseMock: true, MockRegisters: 64, MockCoils: 64})
	if err != nil {
		t.Fatalf("failed to create mock driver: %v", err)
	}
	defer func() { _ = driver.Close() }()

	cs, cleanup := newToolTestClientSession(t, driver, &WritePolicy{enabled: true}, nil)
	defer cleanup()

	tests := []struct {
		name        string
		args        map[string]any
		errContains string
	}{
		{
			name: "missing value",
			args: map[string]any{
				"address":   0,
				"data_type": "uint16",
			},
			errContains: "provide exactly one",
		},
		{
			name: "string type missing quantity",
			args: map[string]any{
				"address":      0,
				"data_type":    "string",
				"string_value": "A",
			},
			errContains: "quantity must be provided for data_type \"string\"",
		},
		{
			name: "string too long",
			args: map[string]any{
				"address":      0,
				"data_type":    "string",
				"quantity":     1,
				"string_value": "ABC",
			},
			errContains: "string too long",
		},
		{
			name: "numeric value with string type",
			args: map[string]any{
				"address":       0,
				"data_type":     "string",
				"quantity":      2,
				"numeric_value": 7,
			},
			errContains: "numeric write is not supported",
		},
		{
			name: "out of range int16",
			args: map[string]any{
				"address":       0,
				"data_type":     "int16",
				"numeric_value": 50000,
			},
			errContains: "out of int16 range",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "write-holding-registers-typed", Arguments: tc.args})
			if err != nil {
				t.Fatalf("tool call failed: %v", err)
			}
			if !res.IsError {
				t.Fatalf("expected error, got success: %s", toolResultText(res))
			}
			if !strings.Contains(toolResultText(res), tc.errContains) {
				t.Fatalf("unexpected error message: %s", toolResultText(res))
			}
		})
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
