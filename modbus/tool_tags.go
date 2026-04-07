package modbus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTagTools(s *mcp.Server, driver Driver, writePolicy *WritePolicy, tagMap *TagMap) {
	mcp.AddTool(s,
		&mcp.Tool{Name: "list-tags", Description: "List configured semantic Modbus tags"},
		func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
			tags := tagMap.List()
			if len(tags) == 0 {
				return successResult("No tags configured. Add tags via CSV or config file."), nil, nil
			}
			raw, err := json.MarshalIndent(tags, "", "  ")
			if err != nil {
				return errorResult(fmt.Sprintf("failed to format tag list: %v", err)), nil, nil
			}
			return successResult(string(raw)), nil, nil
		},
	)

	mcp.AddTool(s,
		&mcp.Tool{Name: "read-tag", Description: "Read a configured semantic Modbus tag"},
		func(ctx context.Context, req *mcp.CallToolRequest, args ReadTagArgs) (*mcp.CallToolResult, any, error) {
			tag, ok := tagMap.Get(args.Name)
			if !ok {
				return errorResult(fmt.Sprintf("tag %q not found", args.Name)), nil, nil
			}
			if !tag.Readable() {
				return errorResult(fmt.Sprintf("tag %q is not readable", tag.Name)), nil, nil
			}

			targetSlave := resolveSlaveID(args.SlaveID, tag.SlaveID)
			return executeTool(ctx, driver, targetSlave, true, func() (*mcp.CallToolResult, error) {
				switch tag.Kind {
				case TagKindHolding:
					results, err := driver.ReadHoldingRegisters(tag.Address, tag.Quantity)
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
					results, err := driver.ReadCoils(tag.Address, tag.Quantity)
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
		&mcp.Tool{Name: "write-tag", Description: "Write a configured semantic Modbus tag"},
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

				return executeTool(ctx, driver, targetSlave, false, func() (*mcp.CallToolResult, error) {
					if len(holdingValues) == 1 {
						_, err := driver.WriteSingleRegister(tag.Address, holdingValues[0])
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
					_, err := driver.WriteMultipleRegisters(tag.Address, uint16(len(holdingValues)), data)
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

				return executeTool(ctx, driver, targetSlave, false, func() (*mcp.CallToolResult, error) {
					byteCount := (len(coilValues) + 7) / 8
					coilBytes := make([]byte, byteCount)
					for i, val := range coilValues {
						if val {
							coilBytes[i/8] |= (1 << uint(i%8))
						}
					}
					_, err := driver.WriteMultipleCoils(tag.Address, uint16(len(coilValues)), coilBytes)
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
