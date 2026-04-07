package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApplyConfigOverridesRespectsFlagPrecedence(t *testing.T) {
	cfg := &AppConfig{
		ModbusIP:      ptr("10.0.0.10"),
		ModbusPort:    ptr(1502),
		ModbusTimeout: ptr("3s"),
		Transport:     ptr("stdio"),
	}

	opts := &RuntimeOptions{
		ModbusDriver:        "goburrow",
		ModbusMode:          "tcp",
		BaudRate:            9600,
		DataBits:            8,
		Parity:              "N",
		StopBits:            1,
		ModbusIP:            "192.168.1.22",
		ModbusPort:          5002,
		ModbusTimeout:       10 * time.Second,
		ModbusIdleTimeout:   2 * time.Second,
		ModbusRetryAttempts: 3,
		ModbusRetryBackoff:  150 * time.Millisecond,
		ModbusRetryOnWrite:  false,
		CircuitTripAfter:    3,
		CircuitOpenFor:      2 * time.Second,
		MockMode:            false,
		MockRegisters:       1024,
		MockCoils:           1024,
		Transport:           "streamable",
	}

	err := ApplyConfigOverrides(cfg, map[string]bool{"modbus-ip": true}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.ModbusIP != "192.168.1.22" {
		t.Fatalf("expected CLI flag value to win for modbus-ip")
	}
	if opts.ModbusPort != 1502 {
		t.Fatalf("expected config to set modbus-port")
	}
	if opts.ModbusTimeout != 3*time.Second {
		t.Fatalf("expected config to set modbus-timeout")
	}
	if opts.Transport != "stdio" {
		t.Fatalf("expected config to set transport")
	}
}

func TestApplyConfigOverridesInvalidDuration(t *testing.T) {
	cfg := &AppConfig{ModbusRetryBackoff: ptr("nope")}

	opts := &RuntimeOptions{
		ModbusDriver:        "goburrow",
		ModbusMode:          "tcp",
		BaudRate:            9600,
		DataBits:            8,
		Parity:              "N",
		StopBits:            1,
		ModbusIP:            "192.168.1.22",
		ModbusPort:          5002,
		ModbusTimeout:       10 * time.Second,
		ModbusIdleTimeout:   2 * time.Second,
		ModbusRetryAttempts: 3,
		ModbusRetryBackoff:  150 * time.Millisecond,
		ModbusRetryOnWrite:  false,
		CircuitTripAfter:    3,
		CircuitOpenFor:      2 * time.Second,
		MockMode:            false,
		MockRegisters:       1024,
		MockCoils:           1024,
		Transport:           "streamable",
	}

	err := ApplyConfigOverrides(cfg, map[string]bool{}, opts)
	if err == nil {
		t.Fatalf("expected invalid duration to fail")
	}
}

func TestToTagMap(t *testing.T) {
	cfg := &AppConfig{
		Tags: []TagConfig{
			{Name: "ambient_temp", Kind: "holding_register", Address: 10, Quantity: 1, Access: "read"},
		},
	}

	tagMap, err := ToTagMap(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tagMap == nil {
		t.Fatalf("expected tag map")
	}
	if _, ok := tagMap.Get("ambient_temp"); !ok {
		t.Fatalf("expected configured tag to exist")
	}
}

func TestToTagMapFromCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.csv")
	csv := "name,kind,address,quantity,access,data_type\nboiler_temp,holding_register,20,2,read,float32\n"
	if err := os.WriteFile(path, []byte(csv), 0644); err != nil {
		t.Fatalf("failed to write temp csv: %v", err)
	}

	tagMap, err := ToTagMap(nil, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tagMap == nil {
		t.Fatalf("expected tag map")
	}
	tag, ok := tagMap.Get("boiler_temp")
	if !ok {
		t.Fatalf("expected CSV tag to exist")
	}
	if tag.DataType != "float32" {
		t.Fatalf("expected data_type float32, got %q", tag.DataType)
	}
}

func TestToTagMapFromCSVDeriveQuantity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.csv")
	csv := "name,kind,address,access,data_type\nboiler_temp,holding_register,20,read,float32\n"
	if err := os.WriteFile(path, []byte(csv), 0644); err != nil {
		t.Fatalf("failed to write temp csv: %v", err)
	}

	tagMap, err := ToTagMap(nil, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tag, ok := tagMap.Get("boiler_temp")
	if !ok {
		t.Fatalf("expected CSV tag to exist")
	}
	if tag.Quantity != 2 {
		t.Fatalf("expected derived quantity 2 for float32, got %d", tag.Quantity)
	}
}

func TestToTagMapFromCSVMissingRequiredColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.csv")
	csv := "name,kind\nboiler_temp,holding_register\n"
	if err := os.WriteFile(path, []byte(csv), 0644); err != nil {
		t.Fatalf("failed to write temp csv: %v", err)
	}

	if _, err := ToTagMap(nil, path); err == nil {
		t.Fatalf("expected missing required column error")
	}
}

func ptr[T any](v T) *T {
	return &v
}
