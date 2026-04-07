package config

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/devidasjadhav/go-mdbus-mcp/modbus"
	"gopkg.in/yaml.v3"
)

var validTransports = map[string]bool{"stdio": true, "sse": true, "streamable": true}
var validDrivers = map[string]bool{"goburrow": true, "simonvetter": true}
var validModes = map[string]bool{"tcp": true, "rtu": true}
var validParities = map[string]bool{"N": true, "E": true, "O": true}

// AppConfig is the raw config-file representation (YAML/JSON).
type AppConfig struct {
	ModbusDriver             *string `json:"modbus_driver" yaml:"modbus_driver"`
	ModbusMode               *string `json:"modbus_mode" yaml:"modbus_mode"`
	SerialPort               *string `json:"serial_port" yaml:"serial_port"`
	BaudRate                 *int    `json:"baud_rate" yaml:"baud_rate"`
	DataBits                 *int    `json:"data_bits" yaml:"data_bits"`
	Parity                   *string `json:"parity" yaml:"parity"`
	StopBits                 *int    `json:"stop_bits" yaml:"stop_bits"`
	ModbusIP                 *string `json:"modbus_ip" yaml:"modbus_ip"`
	ModbusPort               *int    `json:"modbus_port" yaml:"modbus_port"`
	ModbusTimeout            *string `json:"modbus_timeout" yaml:"modbus_timeout"`
	ModbusIdleTimeout        *string `json:"modbus_idle_timeout" yaml:"modbus_idle_timeout"`
	ModbusRetryAttempts      *int    `json:"modbus_retry_attempts" yaml:"modbus_retry_attempts"`
	ModbusRetryBackoff       *string `json:"modbus_retry_backoff" yaml:"modbus_retry_backoff"`
	ModbusRetryOnWrite       *bool   `json:"modbus_retry_on_write" yaml:"modbus_retry_on_write"`
	ModbusReconnectPerOp     *bool   `json:"modbus_reconnect_per_operation" yaml:"modbus_reconnect_per_operation"`
	ModbusConnectionPoolSize *int    `json:"modbus_connection_pool_size" yaml:"modbus_connection_pool_size"`
	CircuitTripAfter         *int    `json:"modbus_circuit_trip_after" yaml:"modbus_circuit_trip_after"`
	CircuitOpenFor           *string `json:"modbus_circuit_open_for" yaml:"modbus_circuit_open_for"`
	Transport                *string `json:"transport" yaml:"transport"`
	MockMode                 *bool   `json:"mock_mode" yaml:"mock_mode"`
	MockRegisters            *int    `json:"mock_registers" yaml:"mock_registers"`
	MockCoils                *int    `json:"mock_coils" yaml:"mock_coils"`

	WritePolicy *WritePolicyConfig `json:"write_policy" yaml:"write_policy"`
	Tags        []TagConfig        `json:"tags" yaml:"tags"`
	TagMapCSV   *string            `json:"tag_map_csv" yaml:"tag_map_csv"`
}

// WritePolicyConfig declares optional write-safety policy overrides from config.
type WritePolicyConfig struct {
	WritesEnabled         *bool   `json:"writes_enabled" yaml:"writes_enabled"`
	WriteAllowlist        *string `json:"write_allowlist" yaml:"write_allowlist"`
	HoldingWriteAllowlist *string `json:"holding_write_allowlist" yaml:"holding_write_allowlist"`
	CoilWriteAllowlist    *string `json:"coil_write_allowlist" yaml:"coil_write_allowlist"`
}

// TagConfig defines one semantic Modbus tag from config.
type TagConfig struct {
	Name        string   `json:"name" yaml:"name"`
	Kind        string   `json:"kind" yaml:"kind"`
	Address     uint16   `json:"address" yaml:"address"`
	Quantity    uint16   `json:"quantity" yaml:"quantity"`
	SlaveID     *uint8   `json:"slave_id,omitempty" yaml:"slave_id,omitempty"`
	Access      string   `json:"access" yaml:"access"`
	DataType    string   `json:"data_type,omitempty" yaml:"data_type,omitempty"`
	ByteOrder   string   `json:"byte_order,omitempty" yaml:"byte_order,omitempty"`
	WordOrder   string   `json:"word_order,omitempty" yaml:"word_order,omitempty"`
	Scale       *float64 `json:"scale,omitempty" yaml:"scale,omitempty"`
	Offset      *float64 `json:"offset,omitempty" yaml:"offset,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

// RuntimeOptions is the fully-resolved runtime configuration after merge.
type RuntimeOptions struct {
	ModbusDriver             string
	ModbusMode               string
	SerialPort               string
	BaudRate                 int
	DataBits                 int
	Parity                   string
	StopBits                 int
	ModbusIP                 string
	ModbusPort               int
	ModbusTimeout            time.Duration
	ModbusIdleTimeout        time.Duration
	ModbusRetryAttempts      int
	ModbusRetryBackoff       time.Duration
	ModbusRetryOnWrite       bool
	ModbusReconnectPerOp     bool
	ModbusConnectionPoolSize int
	CircuitTripAfter         int
	CircuitOpenFor           time.Duration
	MockMode                 bool
	MockRegisters            int
	MockCoils                int
	Transport                string
}

// LoadAppConfig loads YAML/JSON app configuration from disk.
func LoadAppConfig(path string) (*AppConfig, error) {
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

// ApplyConfigOverrides applies config-file values unless a CLI flag explicitly set them.
func ApplyConfigOverrides(cfg *AppConfig, setFlags map[string]bool, opts *RuntimeOptions) error {
	if cfg == nil || opts == nil {
		return nil
	}

	if cfg.ModbusDriver != nil && !setFlags["modbus-driver"] {
		opts.ModbusDriver = *cfg.ModbusDriver
	}
	if cfg.ModbusMode != nil && !setFlags["modbus-mode"] {
		opts.ModbusMode = *cfg.ModbusMode
	}
	if cfg.SerialPort != nil && !setFlags["serial-port"] {
		opts.SerialPort = *cfg.SerialPort
	}
	if cfg.BaudRate != nil && !setFlags["baud-rate"] {
		opts.BaudRate = *cfg.BaudRate
	}
	if cfg.DataBits != nil && !setFlags["data-bits"] {
		opts.DataBits = *cfg.DataBits
	}
	if cfg.Parity != nil && !setFlags["parity"] {
		opts.Parity = *cfg.Parity
	}
	if cfg.StopBits != nil && !setFlags["stop-bits"] {
		opts.StopBits = *cfg.StopBits
	}
	if cfg.ModbusIP != nil && !setFlags["modbus-ip"] {
		opts.ModbusIP = *cfg.ModbusIP
	}
	if cfg.ModbusPort != nil && !setFlags["modbus-port"] {
		opts.ModbusPort = *cfg.ModbusPort
	}
	if cfg.ModbusRetryAttempts != nil && !setFlags["modbus-retry-attempts"] {
		opts.ModbusRetryAttempts = *cfg.ModbusRetryAttempts
	}
	if cfg.ModbusRetryOnWrite != nil && !setFlags["modbus-retry-on-write"] {
		opts.ModbusRetryOnWrite = *cfg.ModbusRetryOnWrite
	}
	if cfg.ModbusReconnectPerOp != nil && !setFlags["modbus-reconnect-per-operation"] {
		opts.ModbusReconnectPerOp = *cfg.ModbusReconnectPerOp
	}
	if cfg.ModbusConnectionPoolSize != nil && !setFlags["modbus-connection-pool-size"] {
		opts.ModbusConnectionPoolSize = *cfg.ModbusConnectionPoolSize
	}
	if cfg.CircuitTripAfter != nil && !setFlags["modbus-circuit-trip-after"] {
		opts.CircuitTripAfter = *cfg.CircuitTripAfter
	}
	if cfg.Transport != nil && !setFlags["transport"] {
		opts.Transport = *cfg.Transport
	}
	if cfg.MockMode != nil && !setFlags["mock-mode"] {
		opts.MockMode = *cfg.MockMode
	}
	if cfg.MockRegisters != nil && !setFlags["mock-registers"] {
		opts.MockRegisters = *cfg.MockRegisters
	}
	if cfg.MockCoils != nil && !setFlags["mock-coils"] {
		opts.MockCoils = *cfg.MockCoils
	}

	if cfg.ModbusTimeout != nil && !setFlags["modbus-timeout"] {
		v, err := time.ParseDuration(*cfg.ModbusTimeout)
		if err != nil {
			return fmt.Errorf("invalid modbus_timeout %q: %w", *cfg.ModbusTimeout, err)
		}
		opts.ModbusTimeout = v
	}
	if cfg.ModbusIdleTimeout != nil && !setFlags["modbus-idle-timeout"] {
		v, err := time.ParseDuration(*cfg.ModbusIdleTimeout)
		if err != nil {
			return fmt.Errorf("invalid modbus_idle_timeout %q: %w", *cfg.ModbusIdleTimeout, err)
		}
		opts.ModbusIdleTimeout = v
	}
	if cfg.ModbusRetryBackoff != nil && !setFlags["modbus-retry-backoff"] {
		v, err := time.ParseDuration(*cfg.ModbusRetryBackoff)
		if err != nil {
			return fmt.Errorf("invalid modbus_retry_backoff %q: %w", *cfg.ModbusRetryBackoff, err)
		}
		opts.ModbusRetryBackoff = v
	}
	if cfg.CircuitOpenFor != nil && !setFlags["modbus-circuit-open-for"] {
		v, err := time.ParseDuration(*cfg.CircuitOpenFor)
		if err != nil {
			return fmt.Errorf("invalid modbus_circuit_open_for %q: %w", *cfg.CircuitOpenFor, err)
		}
		opts.CircuitOpenFor = v
	}

	return nil
}

// ValidateRuntimeOptions normalizes and validates resolved runtime settings.
func ValidateRuntimeOptions(opts *RuntimeOptions) error {
	if opts == nil {
		return nil
	}

	opts.ModbusDriver = strings.ToLower(strings.TrimSpace(opts.ModbusDriver))
	if opts.ModbusDriver == "" {
		opts.ModbusDriver = "goburrow"
	}
	if !validDrivers[opts.ModbusDriver] {
		return fmt.Errorf("invalid modbus driver %q (expected goburrow|simonvetter)", opts.ModbusDriver)
	}

	opts.ModbusMode = strings.ToLower(strings.TrimSpace(opts.ModbusMode))
	if opts.ModbusMode == "" {
		opts.ModbusMode = "tcp"
	}
	if !validModes[opts.ModbusMode] {
		return fmt.Errorf("invalid modbus mode %q (expected tcp|rtu)", opts.ModbusMode)
	}

	opts.Transport = strings.ToLower(strings.TrimSpace(opts.Transport))
	if opts.Transport == "" {
		opts.Transport = "streamable"
	}
	if !validTransports[opts.Transport] {
		return fmt.Errorf("invalid transport %q (expected stdio|sse|streamable)", opts.Transport)
	}

	opts.Parity = strings.ToUpper(strings.TrimSpace(opts.Parity))
	if opts.Parity == "" {
		opts.Parity = "N"
	}
	if !validParities[opts.Parity] {
		return fmt.Errorf("invalid parity %q (expected N|E|O)", opts.Parity)
	}
	if opts.DataBits <= 0 {
		return fmt.Errorf("data bits must be greater than 0")
	}
	if opts.StopBits <= 0 {
		return fmt.Errorf("stop bits must be greater than 0")
	}
	if opts.BaudRate <= 0 {
		return fmt.Errorf("baud rate must be greater than 0")
	}
	if opts.ModbusConnectionPoolSize <= 0 {
		opts.ModbusConnectionPoolSize = 1
	}

	if opts.ModbusMode == "rtu" && strings.TrimSpace(opts.SerialPort) == "" {
		return fmt.Errorf("serial port is required when modbus mode is rtu")
	}

	return nil
}

// ToWritePolicyOverrides maps config-file write policy fields into modbus policy overrides.
func ToWritePolicyOverrides(cfg *AppConfig) *modbus.WritePolicyOverrides {
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

// ToTagMap builds a validated TagMap from inline tags and/or a CSV mapping.
func ToTagMap(cfg *AppConfig, csvPath string) (*modbus.TagMap, error) {
	if strings.TrimSpace(csvPath) != "" {
		tags, err := loadTagsFromCSV(csvPath)
		if err != nil {
			return nil, err
		}
		return modbus.NewTagMap(tags)
	}

	if cfg == nil || len(cfg.Tags) == 0 {
		return nil, nil
	}

	tags := make([]modbus.TagDef, 0, len(cfg.Tags))
	for _, t := range cfg.Tags {
		scale := 1.0
		scaleSet := false
		if t.Scale != nil {
			scale = *t.Scale
			scaleSet = true
		}
		offset := 0.0
		if t.Offset != nil {
			offset = *t.Offset
		}

		tags = append(tags, modbus.TagDef{
			Name:        t.Name,
			Kind:        modbus.TagKind(t.Kind),
			Address:     t.Address,
			Quantity:    t.Quantity,
			SlaveID:     t.SlaveID,
			Access:      modbus.TagAccess(t.Access),
			DataType:    t.DataType,
			ByteOrder:   t.ByteOrder,
			WordOrder:   t.WordOrder,
			Scale:       scale,
			Offset:      offset,
			ScaleSet:    scaleSet,
			Description: t.Description,
		})
	}

	return modbus.NewTagMap(tags)
}

func loadTagsFromCSV(path string) ([]modbus.TagDef, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open tag csv %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read tag csv: %w", err)
	}
	if len(records) < 1 {
		return nil, fmt.Errorf("tag csv is empty")
	}

	headers := map[string]int{}
	for i, h := range records[0] {
		headers[normalizeHeader(h)] = i
	}
	for _, req := range []string{"name", "kind", "address"} {
		if _, ok := headers[req]; !ok {
			return nil, fmt.Errorf("tag csv missing required column %q", req)
		}
	}

	tags := make([]modbus.TagDef, 0, len(records)-1)
	for rowIdx, row := range records[1:] {
		rowNum := rowIdx + 2
		name := cell(row, headers, "name")
		if strings.TrimSpace(name) == "" {
			continue
		}

		address, err := parseUint16CSV(cell(row, headers, "address"))
		if err != nil {
			return nil, fmt.Errorf("row %d invalid address: %w", rowNum, err)
		}

		quantity := uint16(0)
		if qRaw := strings.TrimSpace(cell(row, headers, "quantity")); qRaw != "" {
			q, err := parseUint16CSV(qRaw)
			if err != nil {
				return nil, fmt.Errorf("row %d invalid quantity: %w", rowNum, err)
			}
			quantity = q
		}

		var slaveID *uint8
		if sRaw := strings.TrimSpace(cell(row, headers, "slave_id")); sRaw != "" {
			s, err := strconv.ParseUint(sRaw, 10, 8)
			if err != nil {
				return nil, fmt.Errorf("row %d invalid slave_id: %w", rowNum, err)
			}
			sv := uint8(s)
			slaveID = &sv
		}

		scale := 1.0
		scaleSet := false
		if scaleRaw := strings.TrimSpace(cell(row, headers, "scale")); scaleRaw != "" {
			s, err := strconv.ParseFloat(scaleRaw, 64)
			if err != nil {
				return nil, fmt.Errorf("row %d invalid scale: %w", rowNum, err)
			}
			scale = s
			scaleSet = true
		}

		offset := 0.0
		if offRaw := strings.TrimSpace(cell(row, headers, "offset")); offRaw != "" {
			o, err := strconv.ParseFloat(offRaw, 64)
			if err != nil {
				return nil, fmt.Errorf("row %d invalid offset: %w", rowNum, err)
			}
			offset = o
		}

		tags = append(tags, modbus.TagDef{
			Name:        name,
			Kind:        modbus.TagKind(cellDefault(row, headers, "kind", "holding_register")),
			Address:     address,
			Quantity:    quantity,
			SlaveID:     slaveID,
			Access:      modbus.TagAccess(cellDefault(row, headers, "access", "read")),
			DataType:    cell(row, headers, "data_type"),
			ByteOrder:   cell(row, headers, "byte_order"),
			WordOrder:   cell(row, headers, "word_order"),
			Scale:       scale,
			Offset:      offset,
			ScaleSet:    scaleSet,
			Description: cell(row, headers, "description"),
		})
	}

	return tags, nil
}

func normalizeHeader(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, " ", "_")
	v = strings.ReplaceAll(v, "-", "_")
	return v
}

func cell(row []string, headers map[string]int, key string) string {
	idx, ok := headers[key]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func cellDefault(row []string, headers map[string]int, key string, fallback string) string {
	v := cell(row, headers, key)
	if v == "" {
		return fallback
	}
	return v
}

func parseUint16CSV(raw string) (uint16, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(v), nil
}
