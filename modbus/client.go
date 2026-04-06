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
	ModbusIP       string
	ModbusPort     int
	Timeout        time.Duration
	IdleTimeout    time.Duration
	DefaultSlaveID uint8
}

// NewModbusClient creates a new Modbus client
func NewModbusClient(config *Config) *ModbusClient {
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	if config.IdleTimeout <= 0 {
		// Keep this lower than many PLC/gateway idle timeouts to proactively
		// close local sockets before the remote peer does.
		config.IdleTimeout = 2 * time.Second
	}

	handler := newTCPHandler(config, config.DefaultSlaveID)

	client := modbus.NewClient(handler)
	return &ModbusClient{
		client:  client,
		handler: handler,
		config:  config,
	}
}

func newTCPHandler(config *Config, slaveID uint8) *modbus.TCPClientHandler {
	handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", config.ModbusIP, config.ModbusPort))
	handler.Timeout = config.Timeout
	handler.IdleTimeout = config.IdleTimeout
	handler.SlaveId = slaveID
	handler.Logger = log.Default()
	return handler
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

// Execute performs a thread-safe Modbus operation and refreshes the TCP connection.
//
// Some Modbus servers close idle TCP sessions after a short timeout. Reusing the same
// socket in that case leads to EOF/broken-pipe errors on the next write. To avoid this,
// each operation closes any prior socket, reconnects, sets the requested slave ID, then
// runs the Modbus call on a fresh connection.
func (mc *ModbusClient) Execute(slaveID uint8, operation func() (*mcp.CallToolResult, error)) (*mcp.CallToolResult, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.handler != nil {
		if err := mc.handler.Close(); err != nil {
			log.Printf("modbus: warning: close before reconnect failed: %v", err)
		}
	}

	handler := newTCPHandler(mc.config, slaveID)

	mc.handler = handler
	mc.client = modbus.NewClient(handler)

	if err := handler.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to Modbus server: %w", err)
	}

	return operation()
}

// Ptr is a helper function to get a pointer to a value
func Ptr[T any](v T) *T {
	return &v
}
