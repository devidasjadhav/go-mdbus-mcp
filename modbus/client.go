package modbus

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
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
	connMu  sync.Mutex
	statsMu sync.Mutex
	stats   clientStats
}

type clientStats struct {
	TotalOperations     uint64
	TotalFailures       uint64
	TotalRetries        uint64
	ConsecutiveFailures uint64
	LastError           string
	LastErrorCategory   string
	LastErrorAt         time.Time
	CircuitOpenUntil    time.Time
}

// ClientStatus exposes connection lifecycle stats for diagnostics.
type ClientStatus struct {
	Driver              string     `json:"driver"`
	Mode                string     `json:"mode"`
	TotalOperations     uint64     `json:"total_operations"`
	TotalFailures       uint64     `json:"total_failures"`
	TotalRetries        uint64     `json:"total_retries"`
	ConsecutiveFailures uint64     `json:"consecutive_failures"`
	LastError           string     `json:"last_error,omitempty"`
	LastErrorCategory   string     `json:"last_error_category,omitempty"`
	LastErrorAt         *time.Time `json:"last_error_at,omitempty"`
	CircuitOpenUntil    *time.Time `json:"circuit_open_until,omitempty"`
	CircuitOpen         bool       `json:"circuit_open"`
}

// Config holds the configuration for the Modbus client
type Config struct {
	Driver                   string
	Mode                     string
	SerialPort               string
	BaudRate                 int
	DataBits                 int
	Parity                   string
	StopBits                 int
	ModbusIP                 string
	ModbusPort               int
	Timeout                  time.Duration
	IdleTimeout              time.Duration
	DefaultSlaveID           uint8
	RetryAttempts            int
	RetryBackoff             time.Duration
	RetryOnWrite             bool
	ReconnectPerOp           bool
	ReconnectPerOpConfigured bool
	ConnectionPoolSize       int
	CircuitTripAfter         int
	CircuitOpenFor           time.Duration
	UseMock                  bool
	MockRegisters            int
	MockCoils                int
}

// NewModbusClient creates a new Modbus client
func NewModbusClient(config *Config) *ModbusClient {
	if config.UseMock {
		registerCount := config.MockRegisters
		if registerCount <= 0 {
			registerCount = 1024
		}
		coilCount := config.MockCoils
		if coilCount <= 0 {
			coilCount = 1024
		}

		return &ModbusClient{
			client: newMockClient(registerCount, coilCount),
			config: config,
		}
	}

	applyCommonDefaults(config)

	handler := newTCPHandler(config, config.DefaultSlaveID)

	client := modbus.NewClient(handler)
	return &ModbusClient{
		client:  client,
		handler: handler,
		config:  config,
	}
}

func applyCommonDefaults(config *Config) {
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	if config.IdleTimeout <= 0 {
		// Keep this lower than many PLC/gateway idle timeouts to proactively
		// close local sockets before the remote peer does.
		config.IdleTimeout = 2 * time.Second
	}
	if config.RetryAttempts <= 0 {
		config.RetryAttempts = 3
	}
	if config.RetryBackoff <= 0 {
		config.RetryBackoff = 150 * time.Millisecond
	}
	if !config.ReconnectPerOpConfigured {
		config.ReconnectPerOp = true
	}
	if config.CircuitTripAfter <= 0 {
		config.CircuitTripAfter = 3
	}
	if config.CircuitOpenFor <= 0 {
		config.CircuitOpenFor = 2 * time.Second
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
	mc.connMu.Lock()
	defer mc.connMu.Unlock()
	if mc.config != nil && mc.config.UseMock {
		return nil
	}
	if mc.handler != nil {
		return mc.handler.Close()
	}
	return nil
}

// Client returns the thread-safe underlying modbus client
func (mc *ModbusClient) Client() modbus.Client {
	return mc.client
}

func (mc *ModbusClient) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	return mc.client.ReadHoldingRegisters(address, quantity)
}

func (mc *ModbusClient) DriverName() string {
	return "goburrow"
}

func (mc *ModbusClient) TransportMode() string {
	mode := strings.ToLower(strings.TrimSpace(mc.config.Mode))
	if mode == "" {
		return "tcp"
	}
	return mode
}

func (mc *ModbusClient) ReadInputRegisters(address, quantity uint16) ([]byte, error) {
	return mc.client.ReadInputRegisters(address, quantity)
}

func (mc *ModbusClient) ReadCoils(address, quantity uint16) ([]byte, error) {
	return mc.client.ReadCoils(address, quantity)
}

func (mc *ModbusClient) ReadDiscreteInputs(address, quantity uint16) ([]byte, error) {
	return mc.client.ReadDiscreteInputs(address, quantity)
}

func (mc *ModbusClient) WriteSingleRegister(address, value uint16) ([]byte, error) {
	return mc.client.WriteSingleRegister(address, value)
}

func (mc *ModbusClient) WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error) {
	return mc.client.WriteMultipleRegisters(address, quantity, value)
}

func (mc *ModbusClient) WriteMultipleCoils(address, quantity uint16, value []byte) ([]byte, error) {
	return mc.client.WriteMultipleCoils(address, quantity, value)
}

// Execute performs a thread-safe Modbus operation with retry/circuit protections.
//
// By default it reconnects for each operation to tolerate short idle timeouts from
// gateways/PLCs. When ReconnectPerOp is disabled, it reuses the current connection and
// reconnects only on retry attempts after transient failures.
func (mc *ModbusClient) Execute(ctx context.Context, slaveID uint8, allowRetry bool, operation func() (*mcp.CallToolResult, error)) (*mcp.CallToolResult, error) {
	if mc.config != nil && mc.config.UseMock {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("operation canceled: %w", err)
		}
		res, err := operation()
		if err != nil {
			mc.recordFailure(err)
			return nil, err
		}
		mc.recordSuccess()
		return res, nil
	}

	mc.connMu.Lock()
	defer mc.connMu.Unlock()

	now := time.Now()
	if openUntil, ok := mc.circuitOpenUntil(now); ok {
		return nil, fmt.Errorf("modbus circuit open until %s after repeated failures", openUntil.Format(time.RFC3339))
	}

	attempts := 1
	if allowRetry || mc.config.RetryOnWrite {
		attempts = mc.config.RetryAttempts
	}
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("operation canceled: %w", err)
		}

		forceReconnect := mc.config.ReconnectPerOp || attempt > 1 || mc.handler == nil || mc.client == nil
		if forceReconnect {
			if mc.handler != nil {
				if err := mc.handler.Close(); err != nil {
					log.Printf("modbus: warning: close before reconnect failed: %v", err)
				}
			}

			handler := newTCPHandler(mc.config, slaveID)
			mc.handler = handler
			mc.client = modbus.NewClient(handler)
			if err := handler.Connect(); err != nil {
				lastErr = fmt.Errorf("failed to connect to Modbus server: %w", err)
				if attempt == attempts || !shouldRetryError(lastErr) {
					mc.recordFailure(lastErr)
					return nil, lastErr
				}
				mc.incrementRetries()
				backoff := mc.backoffForAttempt(attempt)
				log.Printf("modbus: transient error (attempt %d/%d), retrying in %s: %v", attempt, attempts, backoff, lastErr)

				mc.connMu.Unlock()
				timer := time.NewTimer(backoff)
				select {
				case <-ctx.Done():
					timer.Stop()
					mc.connMu.Lock()
					return nil, fmt.Errorf("operation canceled during retry backoff: %w", ctx.Err())
				case <-timer.C:
				}
				mc.connMu.Lock()
				continue
			}
		} else {
			mc.handler.SlaveId = slaveID
		}

		res, err := operation()
		if err == nil {
			mc.recordSuccess()
			return res, nil
		}
		lastErr = err

		if attempt == attempts || !shouldRetryError(lastErr) {
			mc.recordFailure(lastErr)
			return nil, lastErr
		}

		if mc.handler != nil {
			_ = mc.handler.Close()
			mc.handler = nil
			mc.client = nil
		}

		mc.incrementRetries()
		backoff := mc.backoffForAttempt(attempt)
		log.Printf("modbus: transient error (attempt %d/%d), retrying in %s: %v", attempt, attempts, backoff, lastErr)

		mc.connMu.Unlock()
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			mc.connMu.Lock()
			return nil, fmt.Errorf("operation canceled during retry backoff: %w", ctx.Err())
		case <-timer.C:
		}
		mc.connMu.Lock()
	}

	mc.recordFailure(lastErr)
	return nil, lastErr
}

// Status returns lifecycle and retry state for diagnostics.
func (mc *ModbusClient) Status() ClientStatus {
	mc.statsMu.Lock()
	defer mc.statsMu.Unlock()

	status := ClientStatus{
		Driver:              mc.DriverName(),
		Mode:                mc.TransportMode(),
		TotalOperations:     mc.stats.TotalOperations,
		TotalFailures:       mc.stats.TotalFailures,
		TotalRetries:        mc.stats.TotalRetries,
		ConsecutiveFailures: mc.stats.ConsecutiveFailures,
		LastError:           mc.stats.LastError,
		LastErrorCategory:   mc.stats.LastErrorCategory,
		CircuitOpen:         !mc.stats.CircuitOpenUntil.IsZero() && time.Now().Before(mc.stats.CircuitOpenUntil),
	}

	if !mc.stats.LastErrorAt.IsZero() {
		v := mc.stats.LastErrorAt
		status.LastErrorAt = &v
	}
	if !mc.stats.CircuitOpenUntil.IsZero() {
		v := mc.stats.CircuitOpenUntil
		status.CircuitOpenUntil = &v
	}

	return status
}

func (mc *ModbusClient) backoffForAttempt(attempt int) time.Duration {
	backoff := mc.config.RetryBackoff
	for i := 1; i < attempt; i++ {
		backoff *= 2
		if backoff > 2*time.Second {
			return 2 * time.Second
		}
	}
	return backoff
}

func (mc *ModbusClient) recordSuccess() {
	mc.statsMu.Lock()
	defer mc.statsMu.Unlock()
	mc.stats.TotalOperations++
	mc.stats.ConsecutiveFailures = 0
	mc.stats.CircuitOpenUntil = time.Time{}
}

func (mc *ModbusClient) recordFailure(err error) {
	mc.statsMu.Lock()
	defer mc.statsMu.Unlock()
	mc.stats.TotalOperations++
	mc.stats.TotalFailures++
	mc.stats.ConsecutiveFailures++
	mc.stats.LastError = err.Error()
	mc.stats.LastErrorCategory = errorCategory(err)
	mc.stats.LastErrorAt = time.Now()
	if int(mc.stats.ConsecutiveFailures) >= mc.config.CircuitTripAfter {
		mc.stats.CircuitOpenUntil = time.Now().Add(mc.config.CircuitOpenFor)
	}
}

func (mc *ModbusClient) incrementRetries() {
	mc.statsMu.Lock()
	defer mc.statsMu.Unlock()
	mc.stats.TotalRetries++
}

func (mc *ModbusClient) circuitOpenUntil(now time.Time) (time.Time, bool) {
	mc.statsMu.Lock()
	defer mc.statsMu.Unlock()
	if mc.stats.CircuitOpenUntil.IsZero() || !now.Before(mc.stats.CircuitOpenUntil) {
		return time.Time{}, false
	}
	return mc.stats.CircuitOpenUntil, true
}

func shouldRetryError(err error) bool {
	if err == nil {
		return false
	}
	if errorsIsTimeout(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	if containsAny(msg,
		"eof",
		"broken pipe",
		"connection reset by peer",
		"use of closed network connection",
		"i/o timeout",
		"timeout",
	) {
		return true
	}
	return false
}

func errorsIsTimeout(err error) bool {
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		return true
	}
	return false
}

func containsAny(msg string, subs ...string) bool {
	for _, s := range subs {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
