package modbus

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMockSoakReadWrite_EnvGuarded(t *testing.T) {
	if os.Getenv("MODBUS_SOAK_TEST") != "1" {
		t.Skip("set MODBUS_SOAK_TEST=1 to run soak harness")
	}

	iterations := 5000
	if raw := os.Getenv("MODBUS_SOAK_ITERATIONS"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("invalid MODBUS_SOAK_ITERATIONS: %v", err)
		}
		if parsed > 0 {
			iterations = parsed
		}
	}

	driver := NewModbusClient(&Config{UseMock: true, MockRegisters: 4096, MockCoils: 4096})
	defer func() { _ = driver.Close() }()

	ctx := context.Background()
	for i := 0; i < iterations; i++ {
		value := uint16(i % 65535)
		_, err := driver.Execute(ctx, 1, true, func() (*mcp.CallToolResult, error) {
			_, err := driver.WriteSingleRegister(10, value)
			if err != nil {
				return nil, err
			}
			data, err := driver.ReadHoldingRegisters(10, 1)
			if err != nil {
				return nil, err
			}
			if len(data) != 2 {
				t.Fatalf("unexpected read length: %d", len(data))
			}
			got := uint16(data[0])<<8 | uint16(data[1])
			if got != value {
				t.Fatalf("soak mismatch: expected=%d got=%d", value, got)
			}
			return nil, nil
		})
		if err != nil {
			t.Fatalf("soak iteration %d failed: %v", i, err)
		}
	}
}
