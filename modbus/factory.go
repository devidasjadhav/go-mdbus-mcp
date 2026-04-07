package modbus

import "strings"

func NewDriver(config *Config) Driver {
	driver := strings.ToLower(strings.TrimSpace(config.Driver))
	if driver == "" {
		driver = "goburrow"
	}

	if config.UseMock {
		return NewModbusClient(config)
	}

	switch driver {
	case "simonvetter":
		if d, err := newSimonvetterDriver(config); err == nil {
			return d
		}
		return NewModbusClient(config)
	default:
		return NewModbusClient(config)
	}
}
