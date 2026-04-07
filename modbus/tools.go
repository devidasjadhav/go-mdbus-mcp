package modbus

import (
	"context"
	"encoding/json"
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

type ReadTagArgs struct {
	Name    string `json:"name" jsonschema:"Configured tag name to read"`
	SlaveID *uint8 `json:"slave_id,omitempty" jsonschema:"Optional Modbus Slave ID override"`
}

type WriteTagArgs struct {
	Name          string   `json:"name" jsonschema:"Configured tag name to write"`
	HoldingValues []uint16 `json:"holding_values,omitempty" jsonschema:"Values for holding-register tags"`
	CoilValues    []bool   `json:"coil_values,omitempty" jsonschema:"Values for coil tags"`
	NumericValue  *float64 `json:"numeric_value,omitempty" jsonschema:"Typed numeric value for holding-register tag"`
	StringValue   *string  `json:"string_value,omitempty" jsonschema:"Typed string value for holding-register string tag"`
	BoolValue     *bool    `json:"bool_value,omitempty" jsonschema:"Typed bool value for single coil tag"`
	SlaveID       *uint8   `json:"slave_id,omitempty" jsonschema:"Optional Modbus Slave ID override"`
}

// RegisterTools registers all available modbus tools to the MCP server.
func RegisterTools(s *mcp.Server, mc *ModbusClient, writePolicy *WritePolicy, tagMap *TagMap) {
	mcp.AddTool(s,
		&mcp.Tool{
			Name:        "read-holding-registers",
			Description: "Read Modbus holding registers",
		},
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
		&mcp.Tool{
			Name:        "read-coils",
			Description: "Read Modbus coils (digital inputs/outputs)",
		},
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
		&mcp.Tool{
			Name:        "write-holding-registers",
			Description: "Write values to Modbus holding registers",
		},
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
		&mcp.Tool{
			Name:        "write-coils",
			Description: "Write values to Modbus coils (digital outputs)",
		},
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

	mcp.AddTool(s,
		&mcp.Tool{
			Name:        "get-modbus-client-status",
			Description: "Get Modbus client retry and connection lifecycle status",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
			raw, err := json.MarshalIndent(mc.Status(), "", "  ")
			if err != nil {
				return errorResult(fmt.Sprintf("failed to format client status: %v", err)), nil, nil
			}
			return successResult(string(raw)), nil, nil
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{
			Name:        "list-tags",
			Description: "List configured semantic Modbus tags",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
			tags := tagMap.List()
			if len(tags) == 0 {
				return successResult("No tags configured. Add tags in server config file under 'tags'."), nil, nil
			}
			raw, err := json.MarshalIndent(tags, "", "  ")
			if err != nil {
				return errorResult(fmt.Sprintf("failed to format tag list: %v", err)), nil, nil
			}
			return successResult(string(raw)), nil, nil
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{
			Name:        "read-tag",
			Description: "Read a configured semantic Modbus tag",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args ReadTagArgs) (*mcp.CallToolResult, any, error) {
			tag, ok := tagMap.Get(args.Name)
			if !ok {
				return errorResult(fmt.Sprintf("tag %q not found", args.Name)), nil, nil
			}
			if !tag.Readable() {
				return errorResult(fmt.Sprintf("tag %q is not readable", tag.Name)), nil, nil
			}

			targetSlave := resolveSlaveID(args.SlaveID, tag.SlaveID)
			return executeTool(ctx, mc, targetSlave, true, func() (*mcp.CallToolResult, error) {
				switch tag.Kind {
				case TagKindHolding:
					results, err := mc.Client().ReadHoldingRegisters(tag.Address, tag.Quantity)
					if err != nil {
						return nil, fmt.Errorf("error reading tag %q: %w", tag.Name, err)
					}
					values := make([]uint16, len(results)/2)
					for i := 0; i < len(results); i += 2 {
						values[i/2] = uint16(results[i])<<8 | uint16(results[i+1])
					}

					decoded, err := decodeHoldingTagValue(tag, values)
					if err != nil {
						return nil, fmt.Errorf("error decoding tag %q: %w", tag.Name, err)
					}

					payload := map[string]any{
						"name":          tag.Name,
						"kind":          tag.Kind,
						"data_type":     tag.DataType,
						"address":       tag.Address,
						"quantity":      tag.Quantity,
						"raw_values":    values,
						"decoded_value": decoded,
					}
					raw, err := json.MarshalIndent(payload, "", "  ")
					if err != nil {
						return nil, fmt.Errorf("error formatting tag %q result: %w", tag.Name, err)
					}
					return successResult(string(raw)), nil

				case TagKindCoil:
					results, err := mc.Client().ReadCoils(tag.Address, tag.Quantity)
					if err != nil {
						return nil, fmt.Errorf("error reading tag %q: %w", tag.Name, err)
					}
					values := make([]bool, tag.Quantity)
					for i := uint16(0); i < tag.Quantity; i++ {
						byteIndex := i / 8
						bitIndex := i % 8
						if byteIndex < uint16(len(results)) {
							values[i] = (results[byteIndex] & (1 << bitIndex)) != 0
						}
					}
					payload := map[string]any{
						"name":          tag.Name,
						"kind":          tag.Kind,
						"data_type":     "bool",
						"address":       tag.Address,
						"quantity":      tag.Quantity,
						"decoded_value": values,
					}
					raw, err := json.MarshalIndent(payload, "", "  ")
					if err != nil {
						return nil, fmt.Errorf("error formatting tag %q result: %w", tag.Name, err)
					}
					return successResult(string(raw)), nil
				}
				return nil, fmt.Errorf("tag %q has unsupported kind %q", tag.Name, tag.Kind)
			})
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{
			Name:        "write-tag",
			Description: "Write a configured semantic Modbus tag",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args WriteTagArgs) (*mcp.CallToolResult, any, error) {
			tag, ok := tagMap.Get(args.Name)
			if !ok {
				return errorResult(fmt.Sprintf("tag %q not found", args.Name)), nil, nil
			}
			if !tag.Writable() {
				return errorResult(fmt.Sprintf("tag %q is not writable", tag.Name)), nil, nil
			}

			targetSlave := resolveSlaveID(args.SlaveID, tag.SlaveID)
			switch tag.Kind {
			case TagKindHolding:
				specified := 0
				if len(args.HoldingValues) > 0 {
					specified++
				}
				if args.NumericValue != nil {
					specified++
				}
				if args.StringValue != nil {
					specified++
				}
				if specified > 1 {
					return errorResult("ambiguous input: provide only one of holding_values, numeric_value, string_value"), nil, nil
				}

				holdingValues := args.HoldingValues
				if len(holdingValues) == 0 {
					switch {
					case args.NumericValue != nil:
						encoded, err := encodeHoldingTagNumericValue(tag, *args.NumericValue)
						if err != nil {
							return errorResult(fmt.Sprintf("failed to encode numeric_value for tag %q: %v", tag.Name, err)), nil, nil
						}
						holdingValues = encoded
					case args.StringValue != nil:
						encoded, err := encodeHoldingTagStringValue(tag, *args.StringValue)
						if err != nil {
							return errorResult(fmt.Sprintf("failed to encode string_value for tag %q: %v", tag.Name, err)), nil, nil
						}
						holdingValues = encoded
					default:
						return errorResult("provide one of holding_values, numeric_value, or string_value for holding_register tag"), nil, nil
					}
				}
				if len(holdingValues) != int(tag.Quantity) {
					return errorResult(fmt.Sprintf("holding_values length must match tag quantity %d", tag.Quantity)), nil, nil
				}
				if err := writePolicy.ValidateHoldingWrite(tag.Address, len(holdingValues)); err != nil {
					return errorResult(err.Error()), nil, nil
				}

				return executeTool(ctx, mc, targetSlave, false, func() (*mcp.CallToolResult, error) {
					if len(holdingValues) == 1 {
						_, err := mc.Client().WriteSingleRegister(tag.Address, holdingValues[0])
						if err != nil {
							return nil, fmt.Errorf("error writing tag %q: %w", tag.Name, err)
						}
						return successResult(fmt.Sprintf("Tag %s written to %d", tag.Name, holdingValues[0])), nil
					}

					data := make([]byte, len(holdingValues)*2)
					for i, val := range holdingValues {
						data[i*2] = byte(val >> 8)
						data[i*2+1] = byte(val & 0xFF)
					}
					_, err := mc.Client().WriteMultipleRegisters(tag.Address, uint16(len(holdingValues)), data)
					if err != nil {
						return nil, fmt.Errorf("error writing tag %q: %w", tag.Name, err)
					}
					return successResult(fmt.Sprintf("Tag %s written: %v", tag.Name, holdingValues)), nil
				})

			case TagKindCoil:
				specified := 0
				if len(args.CoilValues) > 0 {
					specified++
				}
				if args.BoolValue != nil {
					specified++
				}
				if specified > 1 {
					return errorResult("ambiguous input: provide only one of coil_values or bool_value"), nil, nil
				}

				coilValues := args.CoilValues
				if len(coilValues) == 0 {
					if args.BoolValue != nil {
						if tag.Quantity != 1 {
							return errorResult("bool_value can be used only when tag quantity is 1"), nil, nil
						}
						coilValues = []bool{*args.BoolValue}
					} else {
						return errorResult("provide coil_values or bool_value for coil tag"), nil, nil
					}
				}
				if len(coilValues) != int(tag.Quantity) {
					return errorResult(fmt.Sprintf("coil_values length must match tag quantity %d", tag.Quantity)), nil, nil
				}
				if err := writePolicy.ValidateCoilWrite(tag.Address, len(coilValues)); err != nil {
					return errorResult(err.Error()), nil, nil
				}

				return executeTool(ctx, mc, targetSlave, false, func() (*mcp.CallToolResult, error) {
					byteCount := (len(coilValues) + 7) / 8
					coilBytes := make([]byte, byteCount)
					for i, val := range coilValues {
						if val {
							coilBytes[i/8] |= (1 << uint(i%8))
						}
					}
					_, err := mc.Client().WriteMultipleCoils(tag.Address, uint16(len(coilValues)), coilBytes)
					if err != nil {
						return nil, fmt.Errorf("error writing tag %q: %w", tag.Name, err)
					}
					return successResult(fmt.Sprintf("Tag %s written: %v", tag.Name, coilValues)), nil
				})
			}

			return errorResult(fmt.Sprintf("tag %q has unsupported kind %q", tag.Name, tag.Kind)), nil, nil
		},
	)
}

// executeTool is a helper to run thread-safe operations on the Modbus client
// and handle any protocol errors.
func executeTool(ctx context.Context, mc *ModbusClient, slaveID *uint8, allowRetry bool, operation func() (*mcp.CallToolResult, error)) (*mcp.CallToolResult, any, error) {
	targetSlaveID := uint8(1)
	if slaveID != nil {
		targetSlaveID = *slaveID
	}

	res, err := mc.Execute(ctx, targetSlaveID, allowRetry, operation)
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

func resolveSlaveID(requestSlaveID *uint8, tagSlaveID *uint8) *uint8 {
	if requestSlaveID != nil {
		return requestSlaveID
	}
	if tagSlaveID == nil {
		return nil
	}
	v := *tagSlaveID
	return &v
}
