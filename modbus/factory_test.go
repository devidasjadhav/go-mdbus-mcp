package modbus

import "testing"

func TestNewDriverNilConfigFails(t *testing.T) {
	_, err := NewDriver(nil)
	if err == nil {
		t.Fatalf("expected nil config error")
	}
}

func TestNewDriverSelectsSimonvetterForTCP(t *testing.T) {
	d, err := NewDriver(&Config{
		Driver:           "simonvetter",
		Mode:             "tcp",
		ModbusIP:         "127.0.0.1",
		ModbusPort:       502,
		RetryAttempts:    1,
		CircuitTripAfter: 3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := d.(*simonvetterDriver); !ok {
		t.Fatalf("expected simonvetter driver, got %T", d)
	}
}

func TestNewDriverDefaultsToGoburrow(t *testing.T) {
	d, err := NewDriver(&Config{
		Mode:             "tcp",
		ModbusIP:         "127.0.0.1",
		ModbusPort:       502,
		RetryAttempts:    1,
		CircuitTripAfter: 3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := d.(*ModbusClient); !ok {
		t.Fatalf("expected goburrow driver by default, got %T", d)
	}
}

func TestNewDriverFallsBackToMockForMockMode(t *testing.T) {
	d, err := NewDriver(&Config{
		Driver:        "simonvetter",
		Mode:          "tcp",
		UseMock:       true,
		MockRegisters: 64,
		MockCoils:     64,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := d.(*ModbusClient); !ok {
		t.Fatalf("expected mock goburrow-backed driver in mock mode, got %T", d)
	}
}

func TestNewDriverRTUMissingSerialPortFails(t *testing.T) {
	_, err := NewDriver(&Config{Driver: "simonvetter", Mode: "rtu"})
	if err == nil {
		t.Fatalf("expected rtu missing serial port error")
	}
}

func TestNewDriverTCPPoolReturnsPooledDriver(t *testing.T) {
	d, err := NewDriver(&Config{Mode: "tcp", ModbusIP: "127.0.0.1", ModbusPort: 1502, ConnectionPoolSize: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer d.Close()

	if _, ok := d.(interface{ SelectDriverForOp(bool) Driver }); !ok {
		t.Fatalf("expected pooled driver implementation")
	}
}

func TestNewDriverRTUForcesSingleConnection(t *testing.T) {
	d, err := NewDriver(&Config{Mode: "rtu", SerialPort: "/dev/ttyS0", ConnectionPoolSize: 3, Driver: "simonvetter"})
	if err != nil {
		t.Fatalf("unexpected error creating rtu simonvetter driver: %v", err)
	}
	defer d.Close()

	if _, ok := d.(interface{ SelectDriverForOp(bool) Driver }); ok {
		t.Fatalf("did not expect pooled driver for RTU")
	}

	d2, err := NewDriver(&Config{Mode: "rtu", SerialPort: "/dev/ttyS0", ConnectionPoolSize: 3, Driver: "goburrow"})
	if err != nil {
		t.Fatalf("unexpected error creating rtu goburrow driver: %v", err)
	}
	defer d2.Close()
	if _, ok := d2.(interface{ SelectDriverForOp(bool) Driver }); ok {
		t.Fatalf("did not expect pooled driver for RTU goburrow")
	}
}

func TestNormalizeDriverErrorCategories(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "timeout", err: assertErr("i/o timeout"), want: "timeout:"},
		{name: "connection", err: assertErr("connection refused"), want: "connection:"},
		{name: "protocol", err: assertErr("modbus exception"), want: "protocol:"},
		{name: "other", err: assertErr("boom"), want: "other:"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeDriverError(tc.err)
			if got == nil || len(got.Error()) < len(tc.want) || got.Error()[:len(tc.want)] != tc.want {
				t.Fatalf("expected prefix %q, got %v", tc.want, got)
			}
		})
	}
}

func assertErr(msg string) error { return &testErr{msg: msg} }

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }
