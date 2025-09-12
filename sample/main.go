package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/goburrow/modbus"
	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/streamable_http"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func Ptr[T any](v T) *T {
	return &v
}

type Config struct {
	ModbusIP   string
	ModbusPort int
}

type ModbusClient struct {
	client  modbus.Client
	handler *modbus.TCPClientHandler
	config  *Config
}

func NewModbusClient(config *Config) *ModbusClient {
	handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", config.ModbusIP, config.ModbusPort))
	handler.Timeout = 10000000000 // 10 seconds
	handler.SlaveId = 0           // Try slave ID 0
	handler.Logger = log.Default()

	client := modbus.NewClient(handler)
	return &ModbusClient{
		client:  client,
		handler: handler,
		config:  config,
	}
}

func (mc *ModbusClient) Connect() error {
	return mc.handler.Connect()
}

func (mc *ModbusClient) Close() error {
	if mc.handler != nil {
		return mc.handler.Close()
	}
	return nil
}

func (mc *ModbusClient) EnsureConnected() error {
	// Always recreate handler for fresh connection
	mc.handler = modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", mc.config.ModbusIP, mc.config.ModbusPort))
	mc.handler.Timeout = 10000000000 // 10 seconds
	mc.handler.SlaveId = 0
	mc.handler.Logger = log.Default()

	mc.client = modbus.NewClient(mc.handler)

	return mc.Connect()
}

func NewReadHoldingRegistersTool(mc *ModbusClient) fxctx.Tool {
	return fxctx.NewTool(
		&mcp.Tool{
			Name:        "read-holding-registers",
			Description: Ptr("Read Modbus holding registers"),
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"address": {
						"type":        "integer",
						"description": "Starting address to read from",
					},
					"quantity": {
						"type":        "integer",
						"description": "Number of registers to read",
					},
				},
				Required: []string{"address", "quantity"},
			},
		},
		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			// Connect for this operation
			if err := mc.EnsureConnected(); err != nil {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Failed to connect to Modbus server: %v", err),
						},
					},
					IsError: Ptr(true),
				}
			}
			defer mc.Close() // Always close connection after operation

			addressFloat, ok := args["address"].(float64)
			if !ok {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: "Invalid address parameter: must be a number",
						},
					},
					IsError: Ptr(true),
				}
			}
			quantityFloat, ok := args["quantity"].(float64)
			if !ok {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: "Invalid quantity parameter: must be a number",
						},
					},
					IsError: Ptr(true),
				}
			}

			address := uint16(addressFloat)
			quantity := uint16(quantityFloat)

			log.Printf("Reading holding registers: address=%d, quantity=%d", address, quantity)
			results, err := mc.client.ReadHoldingRegisters(address, quantity)
			if err != nil {
				log.Printf("Error reading holding registers: %v", err)
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Error reading holding registers: %v", err),
						},
					},
					IsError: Ptr(true),
				}
			}
			log.Printf("Successfully read %d bytes", len(results))

			// Convert byte array to uint16 values
			values := make([]uint16, len(results)/2)
			for i := 0; i < len(results); i += 2 {
				values[i/2] = uint16(results[i])<<8 | uint16(results[i+1])
			}

			return &mcp.CallToolResult{
				Content: []interface{}{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Holding registers at address %d: %v", address, values),
					},
				},
			}
		},
	)
}

func NewReadCoilsTool(mc *ModbusClient) fxctx.Tool {
	return fxctx.NewTool(
		&mcp.Tool{
			Name:        "read-coils",
			Description: Ptr("Read Modbus coils (digital inputs/outputs)"),
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"address": {
						"type":        "integer",
						"description": "Starting address to read from",
					},
					"quantity": {
						"type":        "integer",
						"description": "Number of coils to read",
					},
				},
				Required: []string{"address", "quantity"},
			},
		},
		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			// Connect for this operation
			if err := mc.EnsureConnected(); err != nil {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Failed to connect to Modbus server: %v", err),
						},
					},
					IsError: Ptr(true),
				}
			}
			defer mc.Close() // Always close connection after operation

			addressFloat, ok := args["address"].(float64)
			if !ok {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: "Invalid address parameter: must be a number",
						},
					},
					IsError: Ptr(true),
				}
			}
			quantityFloat, ok := args["quantity"].(float64)
			if !ok {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: "Invalid quantity parameter: must be a number",
						},
					},
					IsError: Ptr(true),
				}
			}

			address := uint16(addressFloat)
			quantity := uint16(quantityFloat)

			log.Printf("Reading coils: address=%d, quantity=%d", address, quantity)
			results, err := mc.client.ReadCoils(address, quantity)
			if err != nil {
				log.Printf("Error reading coils: %v", err)
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Error reading coils: %v", err),
						},
					},
					IsError: Ptr(true),
				}
			}
			log.Printf("Successfully read %d bytes", len(results))

			// Convert byte array to individual coil states
			// Each byte contains 8 coil states (bits)
			coilStates := make([]bool, quantity)
			for i := uint16(0); i < quantity; i++ {
				byteIndex := i / 8
				bitIndex := i % 8
				if byteIndex < uint16(len(results)) {
					coilStates[i] = (results[byteIndex] & (1 << bitIndex)) != 0
				}
			}

			return &mcp.CallToolResult{
				Content: []interface{}{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Coils at address %d: %v", address, coilStates),
					},
				},
			}
		},
	)
}

func NewWriteHoldingRegistersTool(mc *ModbusClient) fxctx.Tool {
	return fxctx.NewTool(
		&mcp.Tool{
			Name:        "write-holding-registers",
			Description: Ptr("Write values to Modbus holding registers"),
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"address": {
						"type":        "integer",
						"description": "Starting address to write to",
					},
					"values": {
						"type": "array",
						"items": map[string]interface{}{
							"type": "integer",
						},
						"description": "Array of uint16 values to write",
					},
				},
				Required: []string{"address", "values"},
			},
		},
		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			// Connect for this operation
			if err := mc.EnsureConnected(); err != nil {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Failed to connect to Modbus server: %v", err),
						},
					},
					IsError: Ptr(true),
				}
			}
			defer mc.Close() // Always close connection after operation

			addressFloat, ok := args["address"].(float64)
			if !ok {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Invalid address parameter: %v", args["address"]),
						},
					},
					IsError: Ptr(true),
				}
			}

			valuesInterface, ok := args["values"]
			if !ok {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: "Missing values parameter",
						},
					},
					IsError: Ptr(true),
				}
			}

			valuesInterfaceSlice, ok := valuesInterface.([]interface{})
			if !ok {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Invalid values parameter: expected array, got %T", valuesInterface),
						},
					},
					IsError: Ptr(true),
				}
			}

			address := uint16(addressFloat)
			values := make([]uint16, len(valuesInterfaceSlice))

			for i, val := range valuesInterfaceSlice {
				valFloat, ok := val.(float64)
				if !ok {
					return &mcp.CallToolResult{
						Content: []interface{}{
							mcp.TextContent{
								Type: "text",
								Text: fmt.Sprintf("Invalid value at index %d: expected number, got %T", i, val),
							},
						},
						IsError: Ptr(true),
					}
				}
				values[i] = uint16(valFloat)
			}

			log.Printf("Writing holding registers: address=%d, values=%v", address, values)

			// Convert uint16 values to byte array (big-endian)
			data := make([]byte, len(values)*2)
			for i, val := range values {
				data[i*2] = byte(val >> 8)     // High byte
				data[i*2+1] = byte(val & 0xFF) // Low byte
			}

			// Write multiple registers
			_, err := mc.client.WriteMultipleRegisters(address, uint16(len(values)), data)
			if err != nil {
				log.Printf("Error writing holding registers: %v", err)
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Error writing holding registers: %v", err),
						},
					},
					IsError: Ptr(true),
				}
			}

			log.Printf("Successfully wrote %d holding register values", len(values))

			return &mcp.CallToolResult{
				Content: []interface{}{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Successfully wrote %d values to holding registers starting at address %d: %v", len(values), address, values),
					},
				},
			}
		},
	)
}

func NewWriteCoilsTool(mc *ModbusClient) fxctx.Tool {
	return fxctx.NewTool(
		&mcp.Tool{
			Name:        "write-coils",
			Description: Ptr("Write values to Modbus coils (digital outputs)"),
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"address": {
						"type":        "integer",
						"description": "Starting address to write to",
					},
					"values": {
						"type": "array",
						"items": map[string]interface{}{
							"type": "boolean",
						},
						"description": "Array of boolean values to write",
					},
				},
				Required: []string{"address", "values"},
			},
		},
		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			// Connect for this operation
			if err := mc.EnsureConnected(); err != nil {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Failed to connect to Modbus server: %v", err),
						},
					},
					IsError: Ptr(true),
				}
			}
			defer mc.Close() // Always close connection after operation

			addressFloat, ok := args["address"].(float64)
			if !ok {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Invalid address parameter: %v", args["address"]),
						},
					},
					IsError: Ptr(true),
				}
			}

			valuesInterface, ok := args["values"]
			if !ok {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: "Missing values parameter",
						},
					},
					IsError: Ptr(true),
				}
			}

			valuesInterfaceSlice, ok := valuesInterface.([]interface{})
			if !ok {
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Invalid values parameter: expected array, got %T", valuesInterface),
						},
					},
					IsError: Ptr(true),
				}
			}

			address := uint16(addressFloat)
			values := make([]bool, len(valuesInterfaceSlice))

			for i, val := range valuesInterfaceSlice {
				valBool, ok := val.(bool)
				if !ok {
					return &mcp.CallToolResult{
						Content: []interface{}{
							mcp.TextContent{
								Type: "text",
								Text: fmt.Sprintf("Invalid value at index %d: expected boolean, got %T", i, val),
							},
						},
						IsError: Ptr(true),
					}
				}
				values[i] = valBool
			}

			log.Printf("Writing coils: address=%d, values=%v", address, values)

			// Convert boolean array to byte array for Modbus
			byteCount := (len(values) + 7) / 8 // Calculate bytes needed
			coilBytes := make([]byte, byteCount)

			for i, val := range values {
				if val {
					byteIndex := i / 8
					bitIndex := uint(i % 8)
					coilBytes[byteIndex] |= (1 << bitIndex)
				}
			}

			// Write multiple coils
			_, err := mc.client.WriteMultipleCoils(address, uint16(len(values)), coilBytes)
			if err != nil {
				log.Printf("Error writing coils: %v", err)
				return &mcp.CallToolResult{
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Error writing coils: %v", err),
						},
					},
					IsError: Ptr(true),
				}
			}

			log.Printf("Successfully wrote %d coil values", len(values))

			return &mcp.CallToolResult{
				Content: []interface{}{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Successfully wrote %d values to coils starting at address %d: %v", len(values), address, values),
					},
				},
			}
		},
	)
}

func main() {
	// Parse command-line arguments
	modbusIP := flag.String("modbus-ip", "192.168.1.22", "Modbus server IP address")
	modbusPort := flag.Int("modbus-port", 5002, "Modbus server port")
	flag.Parse()

	config := &Config{
		ModbusIP:   *modbusIP,
		ModbusPort: *modbusPort,
	}

	fmt.Printf("Connecting to Modbus server at %s:%d\n", config.ModbusIP, config.ModbusPort)

	modbusClient := NewModbusClient(config)
	fmt.Println("Modbus client initialized - will connect per operation")

	server := app.
		NewBuilder().
		// adding the tools to the app
		WithTool(func() fxctx.Tool { return NewReadHoldingRegistersTool(modbusClient) }).
		WithTool(func() fxctx.Tool { return NewReadCoilsTool(modbusClient) }).
		WithTool(func() fxctx.Tool { return NewWriteHoldingRegistersTool(modbusClient) }).
		WithTool(func() fxctx.Tool { return NewWriteCoilsTool(modbusClient) }).
		WithTool(func() fxctx.Tool { return NewWriteHoldingRegistersTool(modbusClient) }).
		WithTool(func() fxctx.Tool { return NewWriteCoilsTool(modbusClient) }).
		WithServerCapabilities(&mcp.ServerCapabilities{
			Tools: &mcp.ServerCapabilitiesTools{
				ListChanged: Ptr(false),
			},
		}).
		// setting up server
		WithName("modbus-mcp-server").
		WithVersion("0.0.1").
		WithTransport(
			streamable_http.NewTransport(
				streamable_http.Endpoint{
					Hostname: "localhost",
					Port:     8080,
					Path:     "/mcp",
				}),
		).
		// Configuring fx logging to only show errors
		WithFxOptions(
			fx.Provide(func() *zap.Logger {
				cfg := zap.NewDevelopmentConfig()
				cfg.Level.SetLevel(zap.ErrorLevel)
				logger, _ := cfg.Build()
				return logger
			}),
			fx.Option(fx.WithLogger(
				func(logger *zap.Logger) fxevent.Logger {
					return &fxevent.ZapLogger{Logger: logger}
				},
			)),
		)

	err := server.Run()
	if err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server closed")
		} else {
			log.Fatalf("Server error: %v", err)
		}
	}
}
