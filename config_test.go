package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApplyConfigOverrides_RespectsFlagPrecedence(t *testing.T) {
	cfg := &AppConfig{
		ModbusIP:      ptr("10.0.0.10"),
		ModbusPort:    ptr(1502),
		ModbusTimeout: ptr("3s"),
		Transport:     ptr("stdio"),
	}

	modbusIP := ptr("192.168.1.22")
	modbusPort := ptr(5002)
	modbusTimeout := ptr(10 * time.Second)
	modbusIdleTimeout := ptr(2 * time.Second)
	modbusRetryAttempts := ptr(3)
	modbusRetryBackoff := ptr(150 * time.Millisecond)
	modbusRetryOnWrite := ptr(false)
	modbusCircuitTripAfter := ptr(3)
	modbusCircuitOpenFor := ptr(2 * time.Second)
	transport := ptr("streamable")

	setFlags := map[string]bool{"modbus-ip": true}
	err := applyConfigOverrides(
		cfg,
		setFlags,
		modbusIP,
		modbusPort,
		modbusTimeout,
		modbusIdleTimeout,
		modbusRetryAttempts,
		modbusRetryBackoff,
		modbusRetryOnWrite,
		modbusCircuitTripAfter,
		modbusCircuitOpenFor,
		transport,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *modbusIP != "192.168.1.22" {
		t.Fatalf("expected CLI flag value to win for modbus-ip")
	}
	if *modbusPort != 1502 {
		t.Fatalf("expected config to set modbus-port")
	}
	if *modbusTimeout != 3*time.Second {
		t.Fatalf("expected config to set modbus-timeout")
	}
	if *transport != "stdio" {
		t.Fatalf("expected config to set transport")
	}
}

func TestApplyConfigOverrides_InvalidDuration(t *testing.T) {
	cfg := &AppConfig{ModbusRetryBackoff: ptr("nope")}

	modbusIP := ptr("192.168.1.22")
	modbusPort := ptr(5002)
	modbusTimeout := ptr(10 * time.Second)
	modbusIdleTimeout := ptr(2 * time.Second)
	modbusRetryAttempts := ptr(3)
	modbusRetryBackoff := ptr(150 * time.Millisecond)
	modbusRetryOnWrite := ptr(false)
	modbusCircuitTripAfter := ptr(3)
	modbusCircuitOpenFor := ptr(2 * time.Second)
	transport := ptr("streamable")

	err := applyConfigOverrides(
		cfg,
		map[string]bool{},
		modbusIP,
		modbusPort,
		modbusTimeout,
		modbusIdleTimeout,
		modbusRetryAttempts,
		modbusRetryBackoff,
		modbusRetryOnWrite,
		modbusCircuitTripAfter,
		modbusCircuitOpenFor,
		transport,
	)
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

	tagMap, err := toTagMap(cfg, "")
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

func TestToTagMap_FromCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.csv")
	csv := "name,kind,address,quantity,access,data_type\nboiler_temp,holding_register,20,2,read,float32\n"
	if err := os.WriteFile(path, []byte(csv), 0644); err != nil {
		t.Fatalf("failed to write temp csv: %v", err)
	}

	tagMap, err := toTagMap(nil, path)
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

func TestToTagMap_FromCSV_QuantityDerivedFromDataType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.csv")
	csv := "name,kind,address,access,data_type\nboiler_temp,holding_register,20,read,float32\n"
	if err := os.WriteFile(path, []byte(csv), 0644); err != nil {
		t.Fatalf("failed to write temp csv: %v", err)
	}

	tagMap, err := toTagMap(nil, path)
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

func TestToTagMap_FromCSV_MissingRequiredColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.csv")
	csv := "name,kind\nboiler_temp,holding_register\n"
	if err := os.WriteFile(path, []byte(csv), 0644); err != nil {
		t.Fatalf("failed to write temp csv: %v", err)
	}

	if _, err := toTagMap(nil, path); err == nil {
		t.Fatalf("expected missing required column error")
	}
}

func ptr[T any](v T) *T {
	return &v
}
