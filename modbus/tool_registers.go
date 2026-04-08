package modbus

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerDataTools(s *mcp.Server, driver Driver, writePolicy *WritePolicy) {
	mcp.AddTool(s,
		&mcp.Tool{Name: "read-holding-registers", Description: "Read Modbus holding registers"},
		func(ctx context.Context, req *mcp.CallToolRequest, args ReadArgs) (*mcp.CallToolResult, any, error) {
			return executeTool(ctx, driver, args.SlaveID, true, func(d Driver) (*mcp.CallToolResult, error) {
				if args.Quantity == 0 {
					return nil, fmt.Errorf("quantity must be greater than 0")
				}

				log.Printf("Reading holding registers: address=%d, quantity=%d", args.Address, args.Quantity)
				results, err := d.ReadHoldingRegisters(args.Address, args.Quantity)
				if err != nil {
					return nil, fmt.Errorf("error reading holding registers: %w", err)
				}

				values, err := wordsFromBytesStrict(results)
				if err != nil {
					return nil, fmt.Errorf("invalid holding register response: %w", err)
				}

				return successResult(fmt.Sprintf("Holding registers at address %d: %v", args.Address, values)), nil
			})
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{Name: "read-coils", Description: "Read Modbus coils (digital inputs/outputs)"},
		func(ctx context.Context, req *mcp.CallToolRequest, args ReadArgs) (*mcp.CallToolResult, any, error) {
			return executeTool(ctx, driver, args.SlaveID, true, func(d Driver) (*mcp.CallToolResult, error) {
				if args.Quantity == 0 {
					return nil, fmt.Errorf("quantity must be greater than 0")
				}

				log.Printf("Reading coils: address=%d, quantity=%d", args.Address, args.Quantity)
				results, err := d.ReadCoils(args.Address, args.Quantity)
				if err != nil {
					return nil, fmt.Errorf("error reading coils: %w", err)
				}

				coilStates := boolsFromPackedCoils(results, args.Quantity)

				return successResult(fmt.Sprintf("Coils at address %d: %v", args.Address, coilStates)), nil
			})
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{Name: "read-input-registers", Description: "Read Modbus input registers"},
		func(ctx context.Context, req *mcp.CallToolRequest, args ReadArgs) (*mcp.CallToolResult, any, error) {
			return executeTool(ctx, driver, args.SlaveID, true, func(d Driver) (*mcp.CallToolResult, error) {
				if args.Quantity == 0 {
					return nil, fmt.Errorf("quantity must be greater than 0")
				}

				log.Printf("Reading input registers: address=%d, quantity=%d", args.Address, args.Quantity)
				results, err := d.ReadInputRegisters(args.Address, args.Quantity)
				if err != nil {
					return nil, fmt.Errorf("error reading input registers: %w", err)
				}

				values := wordsFromBytes(results)
				return successResult(fmt.Sprintf("Input registers at address %d: %v", args.Address, values)), nil
			})
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{Name: "read-discrete-inputs", Description: "Read Modbus discrete inputs"},
		func(ctx context.Context, req *mcp.CallToolRequest, args ReadArgs) (*mcp.CallToolResult, any, error) {
			return executeTool(ctx, driver, args.SlaveID, true, func(d Driver) (*mcp.CallToolResult, error) {
				if args.Quantity == 0 {
					return nil, fmt.Errorf("quantity must be greater than 0")
				}

				log.Printf("Reading discrete inputs: address=%d, quantity=%d", args.Address, args.Quantity)
				results, err := d.ReadDiscreteInputs(args.Address, args.Quantity)
				if err != nil {
					return nil, fmt.Errorf("error reading discrete inputs: %w", err)
				}

				states := boolsFromPackedCoils(results, args.Quantity)
				return successResult(fmt.Sprintf("Discrete inputs at address %d: %v", args.Address, states)), nil
			})
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{Name: "read-holding-registers-typed", Description: "Read holding registers and decode typed value"},
		func(ctx context.Context, req *mcp.CallToolRequest, args ReadHoldingTypedArgs) (*mcp.CallToolResult, any, error) {
			return executeTool(ctx, driver, args.SlaveID, true, func(d Driver) (*mcp.CallToolResult, error) {
				tag, err := buildTypedHoldingTag("typed_read", args.Address, args.DataType, args.Quantity, args.ByteOrder, args.WordOrder, args.Scale, args.Offset)
				if err != nil {
					return nil, err
				}

				log.Printf("Reading typed holding registers: address=%d, quantity=%d, data_type=%s", args.Address, tag.Quantity, tag.DataType)
				results, err := d.ReadHoldingRegisters(args.Address, tag.Quantity)
				if err != nil {
					return nil, fmt.Errorf("error reading holding registers: %w", err)
				}

				words, err := wordsFromBytesStrict(results)
				if err != nil {
					return nil, fmt.Errorf("invalid holding register response: %w", err)
				}
				decoded, err := decodeHoldingTagValue(tag, words)
				if err != nil {
					return nil, fmt.Errorf("error decoding holding registers: %w", err)
				}

				return successResult(fmt.Sprintf("Typed holding registers at address %d (%s): decoded=%v raw=%v", args.Address, strings.ToLower(tag.DataType), decoded, words)), nil
			})
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{Name: "write-holding-registers-typed", Description: "Write typed value to Modbus holding registers"},
		func(ctx context.Context, req *mcp.CallToolRequest, args WriteHoldingTypedArgs) (*mcp.CallToolResult, any, error) {
			tag, err := buildTypedHoldingTag("typed_write", args.Address, args.DataType, args.Quantity, args.ByteOrder, args.WordOrder, args.Scale, args.Offset)
			if err != nil {
				return errorResult(err.Error()), nil, nil
			}

			specified := 0
			if args.NumericValue != nil {
				specified++
			}
			if args.StringValue != nil {
				specified++
			}
			if specified != 1 {
				return errorResult("provide exactly one of numeric_value or string_value"), nil, nil
			}

			var values []uint16
			switch {
			case args.NumericValue != nil:
				encoded, err := encodeHoldingTagNumericValue(tag, *args.NumericValue)
				if err != nil {
					return errorResult(fmt.Sprintf("failed to encode numeric_value: %v", err)), nil, nil
				}
				values = encoded
			case args.StringValue != nil:
				encoded, err := encodeHoldingTagStringValue(tag, *args.StringValue)
				if err != nil {
					return errorResult(fmt.Sprintf("failed to encode string_value: %v", err)), nil, nil
				}
				values = encoded
			}

			if err := writePolicy.ValidateHoldingWrite(args.Address, len(values)); err != nil {
				return errorResult(err.Error()), nil, nil
			}

			return executeTool(ctx, driver, args.SlaveID, false, func(d Driver) (*mcp.CallToolResult, error) {
				log.Printf("Writing typed holding registers: address=%d, quantity=%d, data_type=%s, values=%v", args.Address, len(values), tag.DataType, values)
				if len(values) == 1 {
					_, err := d.WriteSingleRegister(args.Address, values[0])
					if err != nil {
						return nil, fmt.Errorf("error writing typed holding register: %w", err)
					}
					return successResult(fmt.Sprintf("Successfully wrote typed holding register at address %d (%s): value=%v raw=%v", args.Address, tag.DataType, typedWriteValue(args.NumericValue, args.StringValue), values)), nil
				}

				data := bytesFromWords(values)
				_, err := d.WriteMultipleRegisters(args.Address, uint16(len(values)), data)
				if err != nil {
					return nil, fmt.Errorf("error writing typed holding registers: %w", err)
				}
				return successResult(fmt.Sprintf("Successfully wrote %d typed holding registers at address %d (%s): value=%v raw=%v", len(values), args.Address, tag.DataType, typedWriteValue(args.NumericValue, args.StringValue), values)), nil
			})
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{Name: "write-holding-registers", Description: "Write values to Modbus holding registers"},
		func(ctx context.Context, req *mcp.CallToolRequest, args WriteHoldingRegistersArgs) (*mcp.CallToolResult, any, error) {
			if err := writePolicy.ValidateHoldingWrite(args.Address, len(args.Values)); err != nil {
				return errorResult(err.Error()), nil, nil
			}

			return executeTool(ctx, driver, args.SlaveID, false, func(d Driver) (*mcp.CallToolResult, error) {
				if len(args.Values) == 0 {
					return nil, fmt.Errorf("values must contain at least one register value")
				}

				log.Printf("Writing holding registers: address=%d, values=%v", args.Address, args.Values)
				if len(args.Values) == 1 {
					_, err := d.WriteSingleRegister(args.Address, args.Values[0])
					if err != nil {
						return nil, fmt.Errorf("error writing holding register: %w", err)
					}

					return successResult(fmt.Sprintf("Successfully wrote holding register at address %d: %d", args.Address, args.Values[0])), nil
				}

				data := bytesFromWords(args.Values)

				_, err := d.WriteMultipleRegisters(args.Address, uint16(len(args.Values)), data)
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

			return executeTool(ctx, driver, args.SlaveID, false, func(d Driver) (*mcp.CallToolResult, error) {
				if len(args.Values) == 0 {
					return nil, fmt.Errorf("values must contain at least one coil value")
				}

				log.Printf("Writing coils: address=%d, values=%v", args.Address, args.Values)
				coilBytes := packedCoilsFromBools(args.Values)

				_, err := d.WriteMultipleCoils(args.Address, uint16(len(args.Values)), coilBytes)
				if err != nil {
					return nil, fmt.Errorf("error writing coils: %w", err)
				}

				return successResult(fmt.Sprintf("Successfully wrote %d values to coils starting at address %d: %v", len(args.Values), args.Address, args.Values)), nil
			})
		},
	)
}

func buildTypedHoldingTag(name string, address uint16, dataType string, quantity *uint16, byteOrder *string, wordOrder *string, scale *float64, offset *float64) (TagDef, error) {
	typeName := normalizeDataType(dataType)
	if typeName == "" {
		return TagDef{}, fmt.Errorf("data_type is required")
	}

	qty := expectedQuantity(TagKindHolding, typeName)
	if quantity != nil {
		qty = *quantity
	}
	if qty == 0 {
		return TagDef{}, fmt.Errorf("quantity must be provided for data_type %q", typeName)
	}

	tag := TagDef{
		Name:      name,
		Kind:      TagKindHolding,
		Address:   address,
		Quantity:  qty,
		DataType:  typeName,
		ByteOrder: "big",
		WordOrder: "msw",
		Scale:     1,
		ScaleSet:  true,
	}
	if byteOrder != nil {
		tag.ByteOrder = normalizeByteOrder(*byteOrder)
	}
	if wordOrder != nil {
		tag.WordOrder = normalizeWordOrder(*wordOrder)
	}
	if scale != nil {
		tag.Scale = *scale
	}
	if offset != nil {
		tag.Offset = *offset
	}
	if err := validateDataType(tag); err != nil {
		return TagDef{}, err
	}

	return tag, nil
}

func typedWriteValue(n *float64, s *string) any {
	if n != nil {
		return *n
	}
	if s != nil {
		return *s
	}
	return nil
}
