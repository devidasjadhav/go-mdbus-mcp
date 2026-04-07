package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	svmodbus "github.com/simonvetter/modbus"
)

type simonvetterDriver struct {
	mu     sync.Mutex
	config *Config
	client *svmodbus.ModbusClient
	stats  clientStats
}

func newSimonvetterDriver(config *Config) (*simonvetterDriver, error) {
	applyCommonDefaults(config)
	if config.BaudRate <= 0 {
		config.BaudRate = 9600
	}
	if config.DataBits <= 0 {
		config.DataBits = 8
	}
	if config.StopBits <= 0 {
		config.StopBits = 1
	}
	if strings.TrimSpace(config.Parity) == "" {
		config.Parity = "N"
	}

	if strings.EqualFold(config.Mode, "rtu") {
		if strings.TrimSpace(config.SerialPort) == "" {
			return nil, fmt.Errorf("simonvetter rtu mode requires serial port")
		}
	}
	return &simonvetterDriver{config: config}, nil
}

func (d *simonvetterDriver) Execute(ctx context.Context, slaveID uint8, allowRetry bool, operation func() (*mcp.CallToolResult, error)) (*mcp.CallToolResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if !d.stats.CircuitOpenUntil.IsZero() && now.Before(d.stats.CircuitOpenUntil) {
		return nil, fmt.Errorf("modbus circuit open until %s after repeated failures", d.stats.CircuitOpenUntil.Format(time.RFC3339))
	}

	attempts := 1
	if allowRetry || d.config.RetryOnWrite {
		attempts = d.config.RetryAttempts
	}
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	rtuMode := strings.EqualFold(strings.TrimSpace(d.config.Mode), "rtu")
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("operation canceled: %w", err)
		}

		if !rtuMode {
			if d.client != nil {
				_ = d.client.Close()
				d.client = nil
			}
		}

		if d.client == nil {
			client, err := d.createClient()
			if err != nil {
				lastErr = normalizeDriverError(err)
			} else {
				d.client = client
				if err := d.client.Open(); err != nil {
					lastErr = normalizeDriverError(err)
					d.client = nil
				}
			}
		}

		if d.client != nil {
			if err := d.client.SetUnitId(slaveID); err != nil {
				lastErr = normalizeDriverError(err)
			} else {
				res, opErr := operation()
				if !rtuMode {
					_ = d.client.Close()
					d.client = nil
				}
				if opErr == nil {
					d.recordSuccess()
					return res, nil
				}
				lastErr = normalizeDriverError(opErr)
			}
		}

		if rtuMode && d.client != nil {
			_ = d.client.Close()
			d.client = nil
		}

		if attempt == attempts || !shouldRetryError(lastErr) {
			d.recordFailure(lastErr)
			return nil, lastErr
		}

		d.stats.TotalRetries++
		backoff := d.backoffForAttempt(attempt)
		log.Printf("modbus/simonvetter: transient error (attempt %d/%d), retrying in %s: %v", attempt, attempts, backoff, lastErr)

		d.mu.Unlock()
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			d.mu.Lock()
			return nil, fmt.Errorf("operation canceled during retry backoff: %w", ctx.Err())
		case <-timer.C:
		}
		d.mu.Lock()
	}

	d.recordFailure(lastErr)
	return nil, lastErr
}

func (d *simonvetterDriver) createClient() (*svmodbus.ModbusClient, error) {
	mode := strings.ToLower(strings.TrimSpace(d.config.Mode))
	if mode == "" {
		mode = "tcp"
	}

	url := fmt.Sprintf("tcp://%s:%d", d.config.ModbusIP, d.config.ModbusPort)
	if mode == "rtu" {
		url = fmt.Sprintf("rtu://%s", d.config.SerialPort)
	}

	parity := svmodbus.PARITY_NONE
	switch strings.ToUpper(strings.TrimSpace(d.config.Parity)) {
	case "E":
		parity = svmodbus.PARITY_EVEN
	case "O":
		parity = svmodbus.PARITY_ODD
	}

	conf := &svmodbus.ClientConfiguration{
		URL:      url,
		Timeout:  d.config.Timeout,
		Logger:   log.Default(),
		Speed:    uint(maxInt(d.config.BaudRate, 9600)),
		DataBits: uint(maxInt(d.config.DataBits, 8)),
		Parity:   parity,
		StopBits: uint(maxInt(d.config.StopBits, 1)),
	}

	return svmodbus.NewClient(conf)
}

func (d *simonvetterDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.client != nil {
		err := d.client.Close()
		d.client = nil
		return err
	}
	return nil
}

func (d *simonvetterDriver) Status() ClientStatus {
	d.mu.Lock()
	defer d.mu.Unlock()

	status := ClientStatus{
		TotalOperations:     d.stats.TotalOperations,
		TotalFailures:       d.stats.TotalFailures,
		TotalRetries:        d.stats.TotalRetries,
		ConsecutiveFailures: d.stats.ConsecutiveFailures,
		LastError:           d.stats.LastError,
		CircuitOpen:         !d.stats.CircuitOpenUntil.IsZero() && time.Now().Before(d.stats.CircuitOpenUntil),
	}

	if !d.stats.LastErrorAt.IsZero() {
		v := d.stats.LastErrorAt
		status.LastErrorAt = &v
	}
	if !d.stats.CircuitOpenUntil.IsZero() {
		v := d.stats.CircuitOpenUntil
		status.CircuitOpenUntil = &v
	}

	return status
}

func (d *simonvetterDriver) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	regs, err := d.client.ReadRegisters(address, quantity, svmodbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	return regsToBytes(regs), nil
}

func (d *simonvetterDriver) ReadInputRegisters(address, quantity uint16) ([]byte, error) {
	regs, err := d.client.ReadRegisters(address, quantity, svmodbus.INPUT_REGISTER)
	if err != nil {
		return nil, err
	}
	return regsToBytes(regs), nil
}

func (d *simonvetterDriver) ReadCoils(address, quantity uint16) ([]byte, error) {
	values, err := d.client.ReadCoils(address, quantity)
	if err != nil {
		return nil, err
	}
	return boolsToPackedBytes(values), nil
}

func (d *simonvetterDriver) ReadDiscreteInputs(address, quantity uint16) ([]byte, error) {
	values, err := d.client.ReadDiscreteInputs(address, quantity)
	if err != nil {
		return nil, err
	}
	return boolsToPackedBytes(values), nil
}

func (d *simonvetterDriver) WriteSingleRegister(address, value uint16) ([]byte, error) {
	if err := d.client.WriteRegister(address, value); err != nil {
		return nil, err
	}
	out := make([]byte, 4)
	binary.BigEndian.PutUint16(out[0:2], address)
	binary.BigEndian.PutUint16(out[2:4], value)
	return out, nil
}

func (d *simonvetterDriver) WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error) {
	if int(quantity)*2 != len(value) {
		return nil, fmt.Errorf("invalid register payload length: got %d, expected %d", len(value), int(quantity)*2)
	}
	regs := make([]uint16, quantity)
	for i := range regs {
		regs[i] = binary.BigEndian.Uint16(value[i*2 : i*2+2])
	}
	if err := d.client.WriteRegisters(address, regs); err != nil {
		return nil, err
	}
	out := make([]byte, 4)
	binary.BigEndian.PutUint16(out[0:2], address)
	binary.BigEndian.PutUint16(out[2:4], quantity)
	return out, nil
}

func (d *simonvetterDriver) WriteMultipleCoils(address, quantity uint16, value []byte) ([]byte, error) {
	coils := make([]bool, quantity)
	for i := 0; i < int(quantity); i++ {
		coils[i] = (value[i/8] & (1 << uint(i%8))) != 0
	}
	if err := d.client.WriteCoils(address, coils); err != nil {
		return nil, err
	}
	out := make([]byte, 4)
	binary.BigEndian.PutUint16(out[0:2], address)
	binary.BigEndian.PutUint16(out[2:4], quantity)
	return out, nil
}

func (d *simonvetterDriver) backoffForAttempt(attempt int) time.Duration {
	backoff := d.config.RetryBackoff
	for i := 1; i < attempt; i++ {
		backoff *= 2
		if backoff > 2*time.Second {
			return 2 * time.Second
		}
	}
	return backoff
}

func (d *simonvetterDriver) recordSuccess() {
	d.stats.TotalOperations++
	d.stats.ConsecutiveFailures = 0
	d.stats.CircuitOpenUntil = time.Time{}
}

func (d *simonvetterDriver) recordFailure(err error) {
	d.stats.TotalOperations++
	d.stats.TotalFailures++
	d.stats.ConsecutiveFailures++
	if err != nil {
		d.stats.LastError = err.Error()
	}
	d.stats.LastErrorAt = time.Now()
	if int(d.stats.ConsecutiveFailures) >= d.config.CircuitTripAfter {
		d.stats.CircuitOpenUntil = time.Now().Add(d.config.CircuitOpenFor)
	}
}

func normalizeDriverError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	category := "other"
	switch {
	case containsAny(msg, "timeout", "i/o timeout", "deadline exceeded"):
		category = "timeout"
	case containsAny(msg, "connection refused", "connection reset", "broken pipe", "no such file", "network"):
		category = "connection"
	case containsAny(msg, "modbus", "exception", "crc", "framing"):
		category = "protocol"
	}
	return fmt.Errorf("%s: %w", category, err)
}

func regsToBytes(regs []uint16) []byte {
	out := make([]byte, len(regs)*2)
	for i, v := range regs {
		binary.BigEndian.PutUint16(out[i*2:i*2+2], v)
	}
	return out
}

func boolsToPackedBytes(values []bool) []byte {
	byteCount := (len(values) + 7) / 8
	out := make([]byte, byteCount)
	for i, v := range values {
		if v {
			out[i/8] |= 1 << uint(i%8)
		}
	}
	return out
}

func maxInt(v int, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}
