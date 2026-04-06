package modbus

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReadArgs defines the input schema for reading modbus data.
type ReadArgs struct {
	Address  uint16 `json:"address" jsonschema:"Starting address to read from"`
	Quantity uint16 `json:"quantity" jsonschema:"Number of registers or coils to read"`
	SlaveID  *uint8 `json:"slave_id,omitempty" jsonschema:"Optional Modbus Slave ID (defaults to 1)"`
}

// WriteHoldingRegistersArgs defines the input schema for writing holding registers.
type WriteHoldingRegistersArgs struct {
	Address uint16   `json:"address" jsonschema:"Starting address to write to"`
	Values  []uint16 `json:"values" jsonschema:"Array of uint16 values to write"`
	SlaveID *uint8   `json:"slave_id,omitempty" jsonschema:"Optional Modbus Slave ID (defaults to 1)"`
}

// WriteCoilsArgs defines the input schema for writing coils.
type WriteCoilsArgs struct {
	Address uint16 `json:"address" jsonschema:"Starting address to write to"`
	Values  []bool `json:"values" jsonschema:"Array of boolean values to write"`
	SlaveID *uint8 `json:"slave_id,omitempty" jsonschema:"Optional Modbus Slave ID (defaults to 1)"`
}

// RegisterTools registers all available modbus tools to the MCP server.
func RegisterTools(s *mcp.Server, mc *ModbusClient) {
	mcp.AddTool(s,
		&mcp.Tool{
			Name:        "read-holding-registers",
			Description: "Read Modbus holding registers",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args ReadArgs) (*mcp.CallToolResult, any, error) {
			return executeTool(mc, args.SlaveID, func() (*mcp.CallToolResult, error) {
				log.Printf("Reading holding registers: address=%d, quantity=%d", args.Address, args.Quantity)
				results, err := mc.Client().ReadHoldingRegisters(args.Address, args.Quantity)
				if err != nil {
					return nil, fmt.Errorf("error reading holding registers: %w", err)
				}

				values := make([]uint16, len(results)/2)
				for i := 0; i < len(results); i += 2 {
					values[i/2] = uint16(results[i])<<8 | uint16(results[i+1])
				}

				return successResult(fmt.Sprintf("Holding registers at address %d: %v", args.Address, values)), nil
			})
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{
			Name:        "read-coils",
			Description: "Read Modbus coils (digital inputs/outputs)",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args ReadArgs) (*mcp.CallToolResult, any, error) {
			return executeTool(mc, args.SlaveID, func() (*mcp.CallToolResult, error) {
				log.Printf("Reading coils: address=%d, quantity=%d", args.Address, args.Quantity)
				results, err := mc.Client().ReadCoils(args.Address, args.Quantity)
				if err != nil {
					return nil, fmt.Errorf("error reading coils: %w", err)
				}

				coilStates := make([]bool, args.Quantity)
				for i := uint16(0); i < args.Quantity; i++ {
					byteIndex := i / 8
					bitIndex := i % 8
					if byteIndex < uint16(len(results)) {
						coilStates[i] = (results[byteIndex] & (1 << bitIndex)) != 0
					}
				}

				return successResult(fmt.Sprintf("Coils at address %d: %v", args.Address, coilStates)), nil
			})
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{
			Name:        "write-holding-registers",
			Description: "Write values to Modbus holding registers",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args WriteHoldingRegistersArgs) (*mcp.CallToolResult, any, error) {
			return executeTool(mc, args.SlaveID, func() (*mcp.CallToolResult, error) {
				log.Printf("Writing holding registers: address=%d, values=%v", args.Address, args.Values)
				data := make([]byte, len(args.Values)*2)
				for i, val := range args.Values {
					data[i*2] = byte(val >> 8)
					data[i*2+1] = byte(val & 0xFF)
				}

				_, err := mc.Client().WriteMultipleRegisters(args.Address, uint16(len(args.Values)), data)
				if err != nil {
					return nil, fmt.Errorf("error writing holding registers: %w", err)
				}

				return successResult(fmt.Sprintf("Successfully wrote %d values to holding registers starting at address %d: %v", len(args.Values), args.Address, args.Values)), nil
			})
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{
			Name:        "write-coils",
			Description: "Write values to Modbus coils (digital outputs)",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args WriteCoilsArgs) (*mcp.CallToolResult, any, error) {
			return executeTool(mc, args.SlaveID, func() (*mcp.CallToolResult, error) {
				log.Printf("Writing coils: address=%d, values=%v", args.Address, args.Values)
				byteCount := (len(args.Values) + 7) / 8
				coilBytes := make([]byte, byteCount)
				for i, val := range args.Values {
					if val {
						coilBytes[i/8] |= (1 << uint(i%8))
					}
				}

				_, err := mc.Client().WriteMultipleCoils(args.Address, uint16(len(args.Values)), coilBytes)
				if err != nil {
					return nil, fmt.Errorf("error writing coils: %w", err)
				}

				return successResult(fmt.Sprintf("Successfully wrote %d values to coils starting at address %d: %v", len(args.Values), args.Address, args.Values)), nil
			})
		},
	)
}

// executeTool is a helper to run thread-safe operations on the Modbus client
// and handle any protocol errors.
func executeTool(mc *ModbusClient, slaveID *uint8, operation func() (*mcp.CallToolResult, error)) (*mcp.CallToolResult, any, error) {
	// Set SlaveID right before executing
	if slaveID != nil {
		mc.SetSlaveID(*slaveID)
	} else {
		mc.SetSlaveID(1) // default to 1 if not provided
	}

	res, err := mc.Execute(operation)
	if err != nil {
		// As per the official SDK docs, we return formatting errors directly inside CallToolResult
		// rather than returning a protocol error to avoid hanging the MCP stream.
		return errorResult(err.Error()), nil, nil
	}
	return res, nil, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: msg,
			},
		},
		IsError: true,
	}
}

func successResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}
}
