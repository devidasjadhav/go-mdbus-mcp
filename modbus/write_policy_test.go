package modbus

import "testing"

func TestWritePolicyDisabledByDefault(t *testing.T) {
	t.Setenv(envWritesEnabled, "")
	wp, err := LoadWritePolicy(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := wp.ValidateHoldingWrite(0, 1); err == nil {
		t.Fatalf("expected guarded rejection when writes disabled")
	}
}

func TestWritePolicyAllowlistEnforced(t *testing.T) {
	t.Setenv(envWritesEnabled, "true")
	t.Setenv(envWriteAllowlist, "0-9")

	wp, err := LoadWritePolicy(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := wp.ValidateHoldingWrite(5, 1); err != nil {
		t.Fatalf("expected in-range write to pass, got: %v", err)
	}
	if err := wp.ValidateHoldingWrite(10, 1); err == nil {
		t.Fatalf("expected out-of-range write to fail")
	}
}

func TestWritePolicyPerTypeOverride(t *testing.T) {
	t.Setenv(envWritesEnabled, "true")
	t.Setenv(envWriteAllowlist, "0-9")
	t.Setenv(envWriteAllowlistRegs, "0-20")
	t.Setenv(envWriteAllowlistCoils, "0-5")

	wp, err := LoadWritePolicy(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := wp.ValidateHoldingWrite(15, 1); err != nil {
		t.Fatalf("expected holding override to allow write, got: %v", err)
	}
	if err := wp.ValidateCoilWrite(15, 1); err == nil {
		t.Fatalf("expected coil override to block write")
	}
}

func TestWritePolicyInvalidAllowlistFails(t *testing.T) {
	t.Setenv(envWritesEnabled, "true")
	t.Setenv(envWriteAllowlist, "a-b")

	if _, err := LoadWritePolicy(nil); err == nil {
		t.Fatalf("expected invalid allowlist to fail")
	}
}
