package main

import (
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

	tagMap, err := toTagMap(cfg)
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

func ptr[T any](v T) *T {
	return &v
}
