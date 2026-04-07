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

	start := int(address)
	end := start + int(quantity)
	return packedCoilsFromBools(m.coils[start:end]), nil
}

func (m *mockClient) ReadDiscreteInputs(address, quantity uint16) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateRange(address, quantity, len(m.coils), "discrete inputs"); err != nil {
		return nil, err
	}

	start := int(address)
	end := start + int(quantity)
	return packedCoilsFromBools(m.coils[start:end]), nil
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
	if expected := (int(quantity) + 7) / 8; len(value) != expected {
		return nil, fmt.Errorf("mock: expected %d coil data bytes, got %d", expected, len(value))
	}

	states := boolsFromPackedCoils(value, quantity)
	copy(m.coils[int(address):int(address)+int(quantity)], states)

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

	start := int(address)
	end := start + int(quantity)
	return bytesFromWords(m.registers[start:end]), nil
}

func (m *mockClient) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateRange(address, quantity, len(m.registers), "holding registers"); err != nil {
		return nil, err
	}

	start := int(address)
	end := start + int(quantity)
	return bytesFromWords(m.registers[start:end]), nil
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
	words, err := wordsFromBytesStrict(value)
	if err != nil {
		return nil, fmt.Errorf("mock: invalid register payload: %w", err)
	}
	if int(quantity) != len(words) {
		return nil, fmt.Errorf("mock: expected %d register words, got %d", int(quantity), len(words))
	}
	for i := range words {
		m.registers[int(address)+i] = words[i]
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
