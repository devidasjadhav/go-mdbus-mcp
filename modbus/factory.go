package modbus

import (
	"fmt"
	"strings"
)

func NewDriver(config *Config) (Driver, error) {
	if config == nil {
		return nil, fmt.Errorf("modbus config is required")
	}

	driver := strings.ToLower(strings.TrimSpace(config.Driver))
	if driver == "" {
		driver = "goburrow"
	}

	if config.UseMock {
		return NewModbusClient(config), nil
	}
	if config.ConnectionPoolSize <= 0 {
		config.ConnectionPoolSize = 1
	}
	if strings.EqualFold(strings.TrimSpace(config.Mode), "rtu") {
		config.ConnectionPoolSize = 1
	}

	if config.ConnectionPoolSize > 1 {
		return newPooledDriver(config, func(cfg *Config) (Driver, error) {
			return newSingleDriver(driver, cfg)
		})
	}

	return newSingleDriver(driver, config)
}

func newSingleDriver(driver string, config *Config) (Driver, error) {
	switch strings.ToLower(strings.TrimSpace(driver)) {
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
