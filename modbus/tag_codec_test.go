package modbus

import (
	"math"
	"testing"
)

func TestDecodeHoldingTagValueFloat32(t *testing.T) {
	tag := TagDef{DataType: "float32", ByteOrder: "big", WordOrder: "msw", Scale: 1}
	val, err := decodeHoldingTagValue(tag, []uint16{0x4148, 0x0000}) // 12.5
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fv, ok := val.(float64)
	if !ok {
		t.Fatalf("expected float64 decoded value, got %T", val)
	}
	if math.Abs(fv-12.5) > 0.0001 {
		t.Fatalf("expected 12.5, got %v", fv)
	}
}

func TestDecodeHoldingTagValueString(t *testing.T) {
	tag := TagDef{DataType: "string", ByteOrder: "big", WordOrder: "msw", Scale: 1}
	val, err := decodeHoldingTagValue(tag, []uint16{0x4142, 0x4300}) // "ABC\x00"
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := val.(string)
	if !ok {
		t.Fatalf("expected string decoded value, got %T", val)
	}
	if s != "ABC" {
		t.Fatalf("expected ABC, got %q", s)
	}
}

func TestEncodeHoldingTagNumericValueFloat32(t *testing.T) {
	tag := TagDef{DataType: "float32", ByteOrder: "big", WordOrder: "msw", Quantity: 2, Scale: 1}
	regs, err := encodeHoldingTagNumericValue(tag, 12.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regs) != 2 || regs[0] != 0x4148 || regs[1] != 0x0000 {
		t.Fatalf("unexpected encoded regs: %v", regs)
	}
}

func TestEncodeHoldingTagStringValue(t *testing.T) {
	tag := TagDef{DataType: "string", ByteOrder: "big", Quantity: 4}
	regs, err := encodeHoldingTagStringValue(tag, "ABC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regs) != 4 {
		t.Fatalf("expected 4 registers, got %d", len(regs))
	}
	if regs[0] != 0x4142 || regs[1] != 0x4300 {
		t.Fatalf("unexpected encoded prefix regs: %v", regs[:2])
	}
}
