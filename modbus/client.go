package modbus

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/goburrow/modbus"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ModbusClient handles Modbus TCP connections
type ModbusClient struct {
	client  modbus.Client
	handler *modbus.TCPClientHandler
	config  *Config
	mu      sync.Mutex // Ensures thread safety for concurrent tool calls
}

// Config holds the configuration for the Modbus client
type Config struct {
	ModbusIP   string
	ModbusPort int
}

// NewModbusClient creates a new Modbus client
func NewModbusClient(config *Config) *ModbusClient {
	handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", config.ModbusIP, config.ModbusPort))
	handler.Timeout = 10 * time.Second
	handler.SlaveId = 0 // Common default
	handler.Logger = log.Default()

	client := modbus.NewClient(handler)
	return &ModbusClient{
		client:  client,
		handler: handler,
		config:  config,
	}
}

// Close closes the connection to the Modbus server
func (mc *ModbusClient) Close() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if mc.handler != nil {
		return mc.handler.Close()
	}
	return nil
}

// Client returns the thread-safe underlying modbus client
func (mc *ModbusClient) Client() modbus.Client {
	return mc.client
}

// Execute performs a thread-safe Modbus operation and handles auto-reconnection
func (mc *ModbusClient) Execute(operation func() (*mcp.CallToolResult, error)) (*mcp.CallToolResult, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Ensure connection before executing (goburrow handles reconnect internally, but Connect() forces it)
	if err := mc.handler.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to Modbus server: %w", err)
	}

	return operation()
}

// Ptr is a helper function to get a pointer to a value
func Ptr[T any](v T) *T {
	return &v
}
