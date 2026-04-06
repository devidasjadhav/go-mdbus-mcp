package modbus

import (
	"context"
	"fmt"
	"log"

	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
)

// --- DRY Utilities ---

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []interface{}{
			mcp.TextContent{
				Type: "text",
				Text: msg,
			},
		},
		IsError: Ptr(true),
	}
}

func getUint16Arg(args map[string]interface{}, key string) (uint16, error) {
	val, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("missing %s parameter", key)
	}
	f, ok := val.(float64)
	if !ok {
		return 0, fmt.Errorf("invalid %s parameter: expected number, got %T", key, val)
	}
	return uint16(f), nil
}

func getUint16ArrayArg(args map[string]interface{}, key string) ([]uint16, error) {
	val, ok := args[key]
	if !ok {
		return nil, fmt.Errorf("missing %s parameter", key)
	}
	slice, ok := val.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid %s parameter: expected array, got %T", key, val)
	}
	res := make([]uint16, len(slice))
	for i, v := range slice {
		f, ok := v.(float64)
		if !ok {
			return nil, fmt.Errorf("invalid value at index %d: expected number, got %T", i, v)
		}
		res[i] = uint16(f)
	}
	return res, nil
}

func getBoolArrayArg(args map[string]interface{}, key string) ([]bool, error) {
	val, ok := args[key]
	if !ok {
		return nil, fmt.Errorf("missing %s parameter", key)
	}
	slice, ok := val.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid %s parameter: expected array, got %T", key, val)
	}
	res := make([]bool, len(slice))
	for i, v := range slice {
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid value at index %d: expected boolean, got %T", i, v)
		}
		res[i] = b
	}
	return res, nil
}

func withModbusConnection(mc *ModbusClient, handler func() (*mcp.CallToolResult, error)) *mcp.CallToolResult {
	res, err := mc.Execute(handler)
	if err != nil {
		return errorResult(err.Error())
	}
	return res
}

func successResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []any{
			mcp.TextContent{
				Type: "text",
				Text: text,
			},
		},
	}
}

// --- Tools ---

// NewReadHoldingRegistersTool creates a tool for reading Modbus holding registers
func NewReadHoldingRegistersTool(mc *ModbusClient) fxctx.Tool {
	return fxctx.NewTool(
		&mcp.Tool{
			Name:        "read-holding-registers",
			Description: Ptr("Read Modbus holding registers"),
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"address":  {"type": "integer", "description": "Starting address to read from"},
					"quantity": {"type": "integer", "description": "Number of registers to read"},
				},
				Required: []string{"address", "quantity"},
			},
		},
		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			return withModbusConnection(mc, func() (*mcp.CallToolResult, error) {
				address, err := getUint16Arg(args, "address")
				if err != nil {
					return nil, err
				}

				quantity, err := getUint16Arg(args, "quantity")
				if err != nil {
					return nil, err
				}

				log.Printf("Reading holding registers: address=%d, quantity=%d", address, quantity)
				results, err := mc.client.ReadHoldingRegisters(address, quantity)
				if err != nil {
					log.Printf("Error reading holding registers: %v", err)
					return nil, fmt.Errorf("Error reading holding registers: %v", err)
				}

				log.Printf("Successfully read %d bytes", len(results))
				values := make([]uint16, len(results)/2)
				for i := 0; i < len(results); i += 2 {
					values[i/2] = uint16(results[i])<<8 | uint16(results[i+1])
				}

				return successResult(fmt.Sprintf("Holding registers at address %d: %v", address, values)), nil
			})
		},
	)
}

// NewReadCoilsTool creates a tool for reading Modbus coils
func NewReadCoilsTool(mc *ModbusClient) fxctx.Tool {
	return fxctx.NewTool(
		&mcp.Tool{
			Name:        "read-coils",
			Description: Ptr("Read Modbus coils (digital inputs/outputs)"),
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"address":  {"type": "integer", "description": "Starting address to read from"},
					"quantity": {"type": "integer", "description": "Number of coils to read"},
				},
				Required: []string{"address", "quantity"},
			},
		},
		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			return withModbusConnection(mc, func() (*mcp.CallToolResult, error) {
				address, err := getUint16Arg(args, "address")
				if err != nil {
					return nil, err
				}

				quantity, err := getUint16Arg(args, "quantity")
				if err != nil {
					return nil, err
				}

				log.Printf("Reading coils: address=%d, quantity=%d", address, quantity)
				results, err := mc.client.ReadCoils(address, quantity)
				if err != nil {
					log.Printf("Error reading coils: %v", err)
					return nil, fmt.Errorf("Error reading coils: %v", err)
				}

				log.Printf("Successfully read %d bytes", len(results))
				coilStates := make([]bool, quantity)
				for i := uint16(0); i < quantity; i++ {
					byteIndex := i / 8
					bitIndex := i % 8
					if byteIndex < uint16(len(results)) {
						coilStates[i] = (results[byteIndex] & (1 << bitIndex)) != 0
					}
				}

				return successResult(fmt.Sprintf("Coils at address %d: %v", address, coilStates)), nil
			})
		},
	)
}

// NewWriteHoldingRegistersTool creates a tool for writing to Modbus holding registers
func NewWriteHoldingRegistersTool(mc *ModbusClient) fxctx.Tool {
	return fxctx.NewTool(
		&mcp.Tool{
			Name:        "write-holding-registers",
			Description: Ptr("Write values to Modbus holding registers"),
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"address": {"type": "integer", "description": "Starting address to write to"},
					"values":  {"type": "array", "items": map[string]interface{}{"type": "integer"}, "description": "Array of uint16 values to write"},
				},
				Required: []string{"address", "values"},
			},
		},
		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			return withModbusConnection(mc, func() (*mcp.CallToolResult, error) {
				address, err := getUint16Arg(args, "address")
				if err != nil {
					return nil, err
				}

				values, err := getUint16ArrayArg(args, "values")
				if err != nil {
					return nil, err
				}

				log.Printf("Writing holding registers: address=%d, values=%v", address, values)
				data := make([]byte, len(values)*2)
				for i, val := range values {
					data[i*2] = byte(val >> 8)     // High byte
					data[i*2+1] = byte(val & 0xFF) // Low byte
				}

				_, err = mc.client.WriteMultipleRegisters(address, uint16(len(values)), data)
				if err != nil {
					log.Printf("Error writing holding registers: %v", err)
					return nil, fmt.Errorf("Error writing holding registers: %v", err)
				}

				log.Printf("Successfully wrote %d holding register values", len(values))
				return successResult(fmt.Sprintf("Successfully wrote %d values to holding registers starting at address %d: %v", len(values), address, values)), nil
			})
		},
	)
}

// NewWriteCoilsTool creates a tool for writing to Modbus coils
func NewWriteCoilsTool(mc *ModbusClient) fxctx.Tool {
	return fxctx.NewTool(
		&mcp.Tool{
			Name:        "write-coils",
			Description: Ptr("Write values to Modbus coils (digital outputs)"),
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"address": {"type": "integer", "description": "Starting address to write to"},
					"values":  {"type": "array", "items": map[string]interface{}{"type": "boolean"}, "description": "Array of boolean values to write"},
				},
				Required: []string{"address", "values"},
			},
		},
		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			return withModbusConnection(mc, func() (*mcp.CallToolResult, error) {
				address, err := getUint16Arg(args, "address")
				if err != nil {
					return nil, err
				}

				values, err := getBoolArrayArg(args, "values")
				if err != nil {
					return nil, err
				}

				log.Printf("Writing coils: address=%d, values=%v", address, values)
				byteCount := (len(values) + 7) / 8
				coilBytes := make([]byte, byteCount)
				for i, val := range values {
					if val {
						coilBytes[i/8] |= (1 << uint(i%8))
					}
				}

				_, err = mc.client.WriteMultipleCoils(address, uint16(len(values)), coilBytes)
				if err != nil {
					log.Printf("Error writing coils: %v", err)
					return nil, fmt.Errorf("Error writing coils: %v", err)
				}

				log.Printf("Successfully wrote %d values to coils", len(values))
				return successResult(fmt.Sprintf("Successfully wrote %d values to coils starting at address %d: %v", len(values), address, values)), nil
			})
		},
	)
}
