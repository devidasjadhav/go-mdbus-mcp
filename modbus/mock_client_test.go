package modbus

import (
	"encoding/binary"
	"testing"
)

func TestMockClientHoldingReadWrite(t *testing.T) {
	m := newMockClient(16, 16)
	if _, err := m.WriteSingleRegister(2, 1234); err != nil {
		t.Fatalf("write single register failed: %v", err)
	}

	data := []byte{0x00, 0x0A, 0x00, 0x14}
	if _, err := m.WriteMultipleRegisters(4, 2, data); err != nil {
		t.Fatalf("write multiple registers failed: %v", err)
	}

	out, err := m.ReadHoldingRegisters(2, 4)
	if err != nil {
		t.Fatalf("read holding registers failed: %v", err)
	}
	vals := []uint16{
		binary.BigEndian.Uint16(out[0:2]),
		binary.BigEndian.Uint16(out[2:4]),
		binary.BigEndian.Uint16(out[4:6]),
		binary.BigEndian.Uint16(out[6:8]),
	}
	if vals[0] != 1234 || vals[2] != 10 || vals[3] != 20 {
		t.Fatalf("unexpected values: %v", vals)
	}
}

func TestMockClientCoilReadWrite(t *testing.T) {
	m := newMockClient(16, 16)
	if _, err := m.WriteMultipleCoils(0, 4, []byte{0b00000101}); err != nil {
		t.Fatalf("write coils failed: %v", err)
	}
	out, err := m.ReadCoils(0, 4)
	if err != nil {
		t.Fatalf("read coils failed: %v", err)
	}
	if len(out) == 0 || (out[0]&0b00001111) != 0b00000101 {
		t.Fatalf("unexpected packed coil byte: %08b", out[0])
	}
}
