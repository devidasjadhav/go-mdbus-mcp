package modbus

import (
	"encoding/binary"
	"fmt"
	"sync"
)

type mockClient struct {
	mu        sync.Mutex
	registers []uint16
	coils     []bool
}

func newMockClient(registerCount int, coilCount int) *mockClient {
	return &mockClient{
		registers: make([]uint16, registerCount),
		coils:     make([]bool, coilCount),
	}
}

func (m *mockClient) ReadCoils(address, quantity uint16) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateRange(address, quantity, len(m.coils), "coils"); err != nil {
		return nil, err
	}

	byteCount := (int(quantity) + 7) / 8
	out := make([]byte, byteCount)
	for i := 0; i < int(quantity); i++ {
		if m.coils[int(address)+i] {
			out[i/8] |= 1 << uint(i%8)
		}
	}
	return out, nil
}

func (m *mockClient) ReadDiscreteInputs(address, quantity uint16) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateRange(address, quantity, len(m.coils), "discrete inputs"); err != nil {
		return nil, err
	}

	byteCount := (int(quantity) + 7) / 8
	out := make([]byte, byteCount)
	for i := 0; i < int(quantity); i++ {
		if m.coils[int(address)+i] {
			out[i/8] |= 1 << uint(i%8)
		}
	}
	return out, nil
}

func (m *mockClient) WriteSingleCoil(address, value uint16) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateRange(address, 1, len(m.coils), "coils"); err != nil {
		return nil, err
	}
	m.coils[address] = value == 0xFF00
	return []byte{byte(value >> 8), byte(value)}, nil
}

func (m *mockClient) WriteMultipleCoils(address, quantity uint16, value []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateRange(address, quantity, len(m.coils), "coils"); err != nil {
		return nil, err
	}
	if quantity == 0 {
		return nil, fmt.Errorf("mock: quantity must be > 0")
	}

	for i := 0; i < int(quantity); i++ {
		m.coils[int(address)+i] = (value[i/8] & (1 << uint(i%8))) != 0
	}

	resp := make([]byte, 4)
	binary.BigEndian.PutUint16(resp[0:2], address)
	binary.BigEndian.PutUint16(resp[2:4], quantity)
	return resp, nil
}

func (m *mockClient) ReadInputRegisters(address, quantity uint16) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateRange(address, quantity, len(m.registers), "input registers"); err != nil {
		return nil, err
	}

	out := make([]byte, int(quantity)*2)
	for i := 0; i < int(quantity); i++ {
		binary.BigEndian.PutUint16(out[i*2:i*2+2], m.registers[int(address)+i])
	}
	return out, nil
}

func (m *mockClient) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateRange(address, quantity, len(m.registers), "holding registers"); err != nil {
		return nil, err
	}

	out := make([]byte, int(quantity)*2)
	for i := 0; i < int(quantity); i++ {
		binary.BigEndian.PutUint16(out[i*2:i*2+2], m.registers[int(address)+i])
	}
	return out, nil
}

func (m *mockClient) WriteSingleRegister(address, value uint16) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateRange(address, 1, len(m.registers), "holding registers"); err != nil {
		return nil, err
	}
	m.registers[address] = value
	resp := make([]byte, 4)
	binary.BigEndian.PutUint16(resp[0:2], address)
	binary.BigEndian.PutUint16(resp[2:4], value)
	return resp, nil
}

func (m *mockClient) WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateRange(address, quantity, len(m.registers), "holding registers"); err != nil {
		return nil, err
	}
	if int(quantity)*2 != len(value) {
		return nil, fmt.Errorf("mock: expected %d data bytes, got %d", int(quantity)*2, len(value))
	}
	for i := 0; i < int(quantity); i++ {
		m.registers[int(address)+i] = binary.BigEndian.Uint16(value[i*2 : i*2+2])
	}
	resp := make([]byte, 4)
	binary.BigEndian.PutUint16(resp[0:2], address)
	binary.BigEndian.PutUint16(resp[2:4], quantity)
	return resp, nil
}

func (m *mockClient) ReadWriteMultipleRegisters(readAddress, readQuantity, writeAddress, writeQuantity uint16, value []byte) ([]byte, error) {
	return nil, fmt.Errorf("mock: ReadWriteMultipleRegisters not implemented")
}

func (m *mockClient) MaskWriteRegister(address, andMask, orMask uint16) ([]byte, error) {
	return nil, fmt.Errorf("mock: MaskWriteRegister not implemented")
}

func (m *mockClient) ReadFIFOQueue(address uint16) ([]byte, error) {
	return nil, fmt.Errorf("mock: ReadFIFOQueue not implemented")
}

func (m *mockClient) validateRange(address uint16, quantity uint16, size int, name string) error {
	if quantity == 0 {
		return fmt.Errorf("mock: quantity must be > 0")
	}
	start := int(address)
	end := start + int(quantity)
	if start < 0 || end > size {
		return fmt.Errorf("mock: %s range out of bounds (%d..%d), size=%d", name, start, end-1, size)
	}
	return nil
}
