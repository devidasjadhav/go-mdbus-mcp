package modbus

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Driver defines the operations needed by MCP tools, independent of
// the underlying Modbus library implementation.
type Driver interface {
	DriverName() string
	TransportMode() string
	Execute(ctx context.Context, slaveID uint8, allowRetry bool, operation func() (*mcp.CallToolResult, error)) (*mcp.CallToolResult, error)
	ReadHoldingRegisters(address, quantity uint16) ([]byte, error)
	ReadInputRegisters(address, quantity uint16) ([]byte, error)
	ReadCoils(address, quantity uint16) ([]byte, error)
	ReadDiscreteInputs(address, quantity uint16) ([]byte, error)
	WriteSingleRegister(address, value uint16) ([]byte, error)
	WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error)
	WriteMultipleCoils(address, quantity uint16, value []byte) ([]byte, error)
	Status() ClientStatus
	Close() error
}
