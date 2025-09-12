package modbus

import (
	"fmt"
	"log"

	"github.com/goburrow/modbus"
)

// ModbusClient handles Modbus TCP connections
type ModbusClient struct {
	client  modbus.Client
	handler *modbus.TCPClientHandler
	config  *Config
}

// Config holds the configuration for the Modbus client
type Config struct {
	ModbusIP   string
	ModbusPort int
}

// NewModbusClient creates a new Modbus client
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

// Connect establishes a connection to the Modbus server
func (mc *ModbusClient) Connect() error {
	return mc.handler.Connect()
}

// Close closes the connection to the Modbus server
func (mc *ModbusClient) Close() error {
	if mc.handler != nil {
		return mc.handler.Close()
	}
	return nil
}

// EnsureConnected ensures the client is connected, reconnecting if necessary
func (mc *ModbusClient) EnsureConnected() error {
	// Always recreate handler for fresh connection
	mc.handler = modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", mc.config.ModbusIP, mc.config.ModbusPort))
	mc.handler.Timeout = 10000000000 // 10 seconds
	mc.handler.SlaveId = 0
	mc.handler.Logger = log.Default()

	mc.client = modbus.NewClient(mc.handler)

	return mc.Connect()
}

// Ptr is a helper function to get a pointer to a value
func Ptr[T any](v T) *T {
	return &v
}
