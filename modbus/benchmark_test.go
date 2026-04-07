package modbus

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func BenchmarkMockRead4Registers(b *testing.B) {
	mc := NewModbusClient(&Config{UseMock: true, MockRegisters: 2048, MockCoils: 2048})
	defer mc.Close()

	ctx := context.Background()
	writeData := []byte{0x00, 0x64, 0x00, 0xC8, 0x01, 0x2C, 0x01, 0x90}

	_, err := mc.Execute(ctx, 1, false, func() (*mcp.CallToolResult, error) {
		_, err := mc.Client().WriteMultipleRegisters(0, 4, writeData)
		return nil, err
	})
	if err != nil {
		b.Fatalf("setup write failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mc.Execute(ctx, 1, true, func() (*mcp.CallToolResult, error) {
			_, err := mc.Client().ReadHoldingRegisters(0, 4)
			return nil, err
		})
		if err != nil {
			b.Fatalf("read failed: %v", err)
		}
	}
}

func BenchmarkMockWriteAndReadVerify(b *testing.B) {
	mc := NewModbusClient(&Config{UseMock: true, MockRegisters: 2048, MockCoils: 2048})
	defer mc.Close()

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		value := uint16(i % 65535)
		_, err := mc.Execute(ctx, 1, false, func() (*mcp.CallToolResult, error) {
			_, err := mc.Client().WriteSingleRegister(0, value)
			if err != nil {
				return nil, err
			}
			data, err := mc.Client().ReadHoldingRegisters(0, 1)
			if err != nil {
				return nil, err
			}
			if len(data) != 2 {
				b.Fatalf("unexpected read length: %d", len(data))
			}
			if uint16(data[0])<<8|uint16(data[1]) != value {
				b.Fatalf("verification mismatch")
			}
			return nil, nil
		})
		if err != nil {
			b.Fatalf("write+verify failed: %v", err)
		}
	}
}
