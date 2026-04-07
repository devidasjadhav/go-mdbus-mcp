package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/devidasjadhav/go-mdbus-mcp/modbus"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	ModbusIP            *string `json:"modbus_ip" yaml:"modbus_ip"`
	ModbusPort          *int    `json:"modbus_port" yaml:"modbus_port"`
	ModbusTimeout       *string `json:"modbus_timeout" yaml:"modbus_timeout"`
	ModbusIdleTimeout   *string `json:"modbus_idle_timeout" yaml:"modbus_idle_timeout"`
	ModbusRetryAttempts *int    `json:"modbus_retry_attempts" yaml:"modbus_retry_attempts"`
	ModbusRetryBackoff  *string `json:"modbus_retry_backoff" yaml:"modbus_retry_backoff"`
	ModbusRetryOnWrite  *bool   `json:"modbus_retry_on_write" yaml:"modbus_retry_on_write"`
	CircuitTripAfter    *int    `json:"modbus_circuit_trip_after" yaml:"modbus_circuit_trip_after"`
	CircuitOpenFor      *string `json:"modbus_circuit_open_for" yaml:"modbus_circuit_open_for"`
	Transport           *string `json:"transport" yaml:"transport"`

	WritePolicy *WritePolicyConfig `json:"write_policy" yaml:"write_policy"`
	Tags        []TagConfig        `json:"tags" yaml:"tags"`
}

type WritePolicyConfig struct {
	WritesEnabled         *bool   `json:"writes_enabled" yaml:"writes_enabled"`
	WriteAllowlist        *string `json:"write_allowlist" yaml:"write_allowlist"`
	HoldingWriteAllowlist *string `json:"holding_write_allowlist" yaml:"holding_write_allowlist"`
	CoilWriteAllowlist    *string `json:"coil_write_allowlist" yaml:"coil_write_allowlist"`
}

type TagConfig struct {
	Name     string `json:"name" yaml:"name"`
	Kind     string `json:"kind" yaml:"kind"`
	Address  uint16 `json:"address" yaml:"address"`
	Quantity uint16 `json:"quantity" yaml:"quantity"`
	SlaveID  *uint8 `json:"slave_id,omitempty" yaml:"slave_id,omitempty"`
	Access   string `json:"access" yaml:"access"`
}

func loadAppConfig(path string) (*AppConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg AppConfig
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			if err2 := json.Unmarshal(raw, &cfg); err2 != nil {
				return nil, fmt.Errorf("failed to parse config as YAML or JSON: %v | %v", err, err2)
			}
		}
	}

	return &cfg, nil
}

func applyConfigOverrides(
	cfg *AppConfig,
	setFlags map[string]bool,
	modbusIP *string,
	modbusPort *int,
	modbusTimeout *time.Duration,
	modbusIdleTimeout *time.Duration,
	modbusRetryAttempts *int,
	modbusRetryBackoff *time.Duration,
	modbusRetryOnWrite *bool,
	modbusCircuitTripAfter *int,
	modbusCircuitOpenFor *time.Duration,
	transportFlag *string,
) error {
	if cfg == nil {
		return nil
	}

	if cfg.ModbusIP != nil && !setFlags["modbus-ip"] {
		*modbusIP = *cfg.ModbusIP
	}
	if cfg.ModbusPort != nil && !setFlags["modbus-port"] {
		*modbusPort = *cfg.ModbusPort
	}
	if cfg.ModbusRetryAttempts != nil && !setFlags["modbus-retry-attempts"] {
		*modbusRetryAttempts = *cfg.ModbusRetryAttempts
	}
	if cfg.ModbusRetryOnWrite != nil && !setFlags["modbus-retry-on-write"] {
		*modbusRetryOnWrite = *cfg.ModbusRetryOnWrite
	}
	if cfg.CircuitTripAfter != nil && !setFlags["modbus-circuit-trip-after"] {
		*modbusCircuitTripAfter = *cfg.CircuitTripAfter
	}
	if cfg.Transport != nil && !setFlags["transport"] {
		*transportFlag = *cfg.Transport
	}

	if cfg.ModbusTimeout != nil && !setFlags["modbus-timeout"] {
		v, err := time.ParseDuration(*cfg.ModbusTimeout)
		if err != nil {
			return fmt.Errorf("invalid modbus_timeout %q: %w", *cfg.ModbusTimeout, err)
		}
		*modbusTimeout = v
	}
	if cfg.ModbusIdleTimeout != nil && !setFlags["modbus-idle-timeout"] {
		v, err := time.ParseDuration(*cfg.ModbusIdleTimeout)
		if err != nil {
			return fmt.Errorf("invalid modbus_idle_timeout %q: %w", *cfg.ModbusIdleTimeout, err)
		}
		*modbusIdleTimeout = v
	}
	if cfg.ModbusRetryBackoff != nil && !setFlags["modbus-retry-backoff"] {
		v, err := time.ParseDuration(*cfg.ModbusRetryBackoff)
		if err != nil {
			return fmt.Errorf("invalid modbus_retry_backoff %q: %w", *cfg.ModbusRetryBackoff, err)
		}
		*modbusRetryBackoff = v
	}
	if cfg.CircuitOpenFor != nil && !setFlags["modbus-circuit-open-for"] {
		v, err := time.ParseDuration(*cfg.CircuitOpenFor)
		if err != nil {
			return fmt.Errorf("invalid modbus_circuit_open_for %q: %w", *cfg.CircuitOpenFor, err)
		}
		*modbusCircuitOpenFor = v
	}

	return nil
}

func toWritePolicyOverrides(cfg *AppConfig) *modbus.WritePolicyOverrides {
	if cfg == nil || cfg.WritePolicy == nil {
		return nil
	}
	return &modbus.WritePolicyOverrides{
		WritesEnabled:         cfg.WritePolicy.WritesEnabled,
		WriteAllowlist:        cfg.WritePolicy.WriteAllowlist,
		HoldingWriteAllowlist: cfg.WritePolicy.HoldingWriteAllowlist,
		CoilWriteAllowlist:    cfg.WritePolicy.CoilWriteAllowlist,
	}
}

func toTagMap(cfg *AppConfig) (*modbus.TagMap, error) {
	if cfg == nil || len(cfg.Tags) == 0 {
		return nil, nil
	}

	tags := make([]modbus.TagDef, 0, len(cfg.Tags))
	for _, t := range cfg.Tags {
		tags = append(tags, modbus.TagDef{
			Name:     t.Name,
			Kind:     modbus.TagKind(t.Kind),
			Address:  t.Address,
			Quantity: t.Quantity,
			SlaveID:  t.SlaveID,
			Access:   modbus.TagAccess(t.Access),
		})
	}

	return modbus.NewTagMap(tags)
}
