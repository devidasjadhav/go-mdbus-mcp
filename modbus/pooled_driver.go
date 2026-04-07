package modbus

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type pooledDriver struct {
	members []Driver
	next    uint32
}

func newPooledDriver(base *Config, create func(*Config) (Driver, error)) (Driver, error) {
	if base == nil {
		return nil, fmt.Errorf("modbus config is required")
	}
	if base.ConnectionPoolSize <= 1 {
		return create(base)
	}

	members := make([]Driver, 0, base.ConnectionPoolSize)
	for i := 0; i < base.ConnectionPoolSize; i++ {
		cfg := *base
		cfg.ConnectionPoolSize = 1
		d, err := create(&cfg)
		if err != nil {
			for _, created := range members {
				_ = created.Close()
			}
			return nil, fmt.Errorf("create pooled member %d: %w", i, err)
		}
		members = append(members, d)
	}

	return &pooledDriver{members: members}, nil
}

func (p *pooledDriver) selectDriver(allowRetry bool) Driver {
	if len(p.members) == 0 {
		return nil
	}
	if !allowRetry {
		return p.members[0]
	}
	idx := int(atomic.AddUint32(&p.next, 1)-1) % len(p.members)
	return p.members[idx]
}

func (p *pooledDriver) DriverName() string {
	if len(p.members) == 0 {
		return "pooled"
	}
	return fmt.Sprintf("%s-pooled", p.members[0].DriverName())
}

func (p *pooledDriver) TransportMode() string {
	if len(p.members) == 0 {
		return "tcp"
	}
	return p.members[0].TransportMode()
}

func (p *pooledDriver) Execute(ctx context.Context, slaveID uint8, allowRetry bool, operation func() (*mcp.CallToolResult, error)) (*mcp.CallToolResult, error) {
	d := p.selectDriver(allowRetry)
	if d == nil {
		return nil, fmt.Errorf("modbus pooled driver has no active members")
	}
	return d.Execute(ctx, slaveID, allowRetry, operation)
}

func (p *pooledDriver) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	return p.selectDriver(true).ReadHoldingRegisters(address, quantity)
}

func (p *pooledDriver) ReadInputRegisters(address, quantity uint16) ([]byte, error) {
	return p.selectDriver(true).ReadInputRegisters(address, quantity)
}

func (p *pooledDriver) ReadCoils(address, quantity uint16) ([]byte, error) {
	return p.selectDriver(true).ReadCoils(address, quantity)
}

func (p *pooledDriver) ReadDiscreteInputs(address, quantity uint16) ([]byte, error) {
	return p.selectDriver(true).ReadDiscreteInputs(address, quantity)
}

func (p *pooledDriver) WriteSingleRegister(address, value uint16) ([]byte, error) {
	return p.selectDriver(false).WriteSingleRegister(address, value)
}

func (p *pooledDriver) WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error) {
	return p.selectDriver(false).WriteMultipleRegisters(address, quantity, value)
}

func (p *pooledDriver) WriteMultipleCoils(address, quantity uint16, value []byte) ([]byte, error) {
	return p.selectDriver(false).WriteMultipleCoils(address, quantity, value)
}

func (p *pooledDriver) Status() ClientStatus {
	if len(p.members) == 0 {
		return ClientStatus{Driver: "pooled", Mode: "tcp"}
	}

	combined := p.members[0].Status()
	for i := 1; i < len(p.members); i++ {
		s := p.members[i].Status()
		combined.TotalOperations += s.TotalOperations
		combined.TotalFailures += s.TotalFailures
		combined.TotalRetries += s.TotalRetries
		combined.ConsecutiveFailures += s.ConsecutiveFailures
		if s.LastErrorAt != nil {
			if combined.LastErrorAt == nil || s.LastErrorAt.After(*combined.LastErrorAt) {
				combined.LastErrorAt = s.LastErrorAt
				combined.LastError = s.LastError
				combined.LastErrorCategory = s.LastErrorCategory
			}
		}
		if s.CircuitOpen {
			combined.CircuitOpen = true
			if s.CircuitOpenUntil != nil {
				if combined.CircuitOpenUntil == nil || s.CircuitOpenUntil.After(*combined.CircuitOpenUntil) {
					combined.CircuitOpenUntil = s.CircuitOpenUntil
				}
			}
		}
	}
	combined.Driver = p.DriverName()
	combined.Mode = p.TransportMode()
	return combined
}

func (p *pooledDriver) Close() error {
	var firstErr error
	for _, d := range p.members {
		if err := d.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (p *pooledDriver) SelectDriverForOp(allowRetry bool) Driver {
	return p.selectDriver(allowRetry)
}
