package modbus

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerDataTools(s *mcp.Server, mc *ModbusClient, writePolicy *WritePolicy) {
	mcp.AddTool(s,
		&mcp.Tool{Name: "read-holding-registers", Description: "Read Modbus holding registers"},
		func(ctx context.Context, req *mcp.CallToolRequest, args ReadArgs) (*mcp.CallToolResult, any, error) {
			return executeTool(ctx, mc, args.SlaveID, true, func() (*mcp.CallToolResult, error) {
				if args.Quantity == 0 {
					return nil, fmt.Errorf("quantity must be greater than 0")
				}

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
		&mcp.Tool{Name: "read-coils", Description: "Read Modbus coils (digital inputs/outputs)"},
		func(ctx context.Context, req *mcp.CallToolRequest, args ReadArgs) (*mcp.CallToolResult, any, error) {
			return executeTool(ctx, mc, args.SlaveID, true, func() (*mcp.CallToolResult, error) {
				if args.Quantity == 0 {
					return nil, fmt.Errorf("quantity must be greater than 0")
				}

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
		&mcp.Tool{Name: "write-holding-registers", Description: "Write values to Modbus holding registers"},
		func(ctx context.Context, req *mcp.CallToolRequest, args WriteHoldingRegistersArgs) (*mcp.CallToolResult, any, error) {
			if err := writePolicy.ValidateHoldingWrite(args.Address, len(args.Values)); err != nil {
				return errorResult(err.Error()), nil, nil
			}

			return executeTool(ctx, mc, args.SlaveID, false, func() (*mcp.CallToolResult, error) {
				if len(args.Values) == 0 {
					return nil, fmt.Errorf("values must contain at least one register value")
				}

				log.Printf("Writing holding registers: address=%d, values=%v", args.Address, args.Values)
				if len(args.Values) == 1 {
					_, err := mc.Client().WriteSingleRegister(args.Address, args.Values[0])
					if err != nil {
						return nil, fmt.Errorf("error writing holding register: %w", err)
					}

					return successResult(fmt.Sprintf("Successfully wrote holding register at address %d: %d", args.Address, args.Values[0])), nil
				}

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
		&mcp.Tool{Name: "write-coils", Description: "Write values to Modbus coils (digital outputs)"},
		func(ctx context.Context, req *mcp.CallToolRequest, args WriteCoilsArgs) (*mcp.CallToolResult, any, error) {
			if err := writePolicy.ValidateCoilWrite(args.Address, len(args.Values)); err != nil {
				return errorResult(err.Error()), nil, nil
			}

			return executeTool(ctx, mc, args.SlaveID, false, func() (*mcp.CallToolResult, error) {
				if len(args.Values) == 0 {
					return nil, fmt.Errorf("values must contain at least one coil value")
				}

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
