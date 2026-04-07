package modbus

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRTUSimonvetterReadHoldingRegisters_EnvGuarded(t *testing.T) {
	port := os.Getenv("MODBUS_RTU_TEST_PORT")
	if port == "" {
		t.Skip("set MODBUS_RTU_TEST_PORT to run RTU integration tests")
	}

	baud := 9600
	if raw := os.Getenv("MODBUS_RTU_TEST_BAUD"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("invalid MODBUS_RTU_TEST_BAUD: %v", err)
		}
		baud = parsed
	}

	unitID := uint8(1)
	if raw := os.Getenv("MODBUS_RTU_TEST_SLAVE_ID"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("invalid MODBUS_RTU_TEST_SLAVE_ID: %v", err)
		}
		unitID = uint8(parsed)
	}

	driver, err := newSimonvetterDriver(&Config{
		Driver:           "simonvetter",
		Mode:             "rtu",
		SerialPort:       port,
		BaudRate:         baud,
		DataBits:         8,
		Parity:           "N",
		StopBits:         1,
		Timeout:          2 * time.Second,
		RetryAttempts:    1,
		CircuitTripAfter: 3,
		CircuitOpenFor:   1 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create simonvetter RTU driver: %v", err)
	}
	defer func() { _ = driver.Close() }()

	_, err = driver.Execute(context.Background(), unitID, true, func() (*mcp.CallToolResult, error) {
		_, e := driver.ReadHoldingRegisters(0, 1)
		return nil, e
	})
	if err != nil {
		t.Fatalf("rtu read failed: %v", err)
	}
}
