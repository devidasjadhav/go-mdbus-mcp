package modbus

import "strings"

func NewDriver(config *Config) (Driver, error) {
	driver := strings.ToLower(strings.TrimSpace(config.Driver))
	if driver == "" {
		driver = "goburrow"
	}

	if config.UseMock {
		return NewModbusClient(config), nil
	}

	switch driver {
	case "simonvetter":
		d, err := newSimonvetterDriver(config)
		if err != nil {
			return nil, err
		}
		return d, nil
	default:
		return NewModbusClient(config), nil
	}
}
