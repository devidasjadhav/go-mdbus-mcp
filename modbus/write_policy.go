package modbus

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	envWritesEnabled       = "MODBUS_WRITES_ENABLED"
	envWriteAllowlist      = "MODBUS_WRITE_ALLOWLIST"
	envWriteAllowlistCoils = "MODBUS_WRITE_ALLOWLIST_COILS"
	envWriteAllowlistRegs  = "MODBUS_WRITE_ALLOWLIST_HOLDING"
)

type addressRange struct {
	start uint16
	end   uint16
}

// WritePolicy controls if write tools are allowed to execute.
type WritePolicy struct {
	enabled      bool
	holdingRange []addressRange
	coilsRange   []addressRange
}

// WritePolicyOverrides allows config-file values to override environment policy.
type WritePolicyOverrides struct {
	WritesEnabled         *bool
	WriteAllowlist        *string
	HoldingWriteAllowlist *string
	CoilWriteAllowlist    *string
}

// LoadWritePolicyFromEnv loads write-safety policy from environment variables.
func LoadWritePolicyFromEnv() (*WritePolicy, error) {
	return LoadWritePolicy(nil)
}

// LoadWritePolicy loads write-safety policy from env and optional overrides.
func LoadWritePolicy(overrides *WritePolicyOverrides) (*WritePolicy, error) {
	enabled := parseEnvBool(envWritesEnabled, false)
	if overrides != nil && overrides.WritesEnabled != nil {
		enabled = *overrides.WritesEnabled
	}

	globalRaw := os.Getenv(envWriteAllowlist)
	if overrides != nil && overrides.WriteAllowlist != nil {
		globalRaw = *overrides.WriteAllowlist
	}
	globalRanges, err := parseAllowlist(globalRaw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", envWriteAllowlist, err)
	}

	holdingRanges := globalRanges
	holdingRaw := os.Getenv(envWriteAllowlistRegs)
	if overrides != nil && overrides.HoldingWriteAllowlist != nil {
		holdingRaw = *overrides.HoldingWriteAllowlist
	}
	if raw := strings.TrimSpace(holdingRaw); raw != "" {
		holdingRanges, err = parseAllowlist(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", envWriteAllowlistRegs, err)
		}
	}

	coilRanges := globalRanges
	coilRaw := os.Getenv(envWriteAllowlistCoils)
	if overrides != nil && overrides.CoilWriteAllowlist != nil {
		coilRaw = *overrides.CoilWriteAllowlist
	}
	if raw := strings.TrimSpace(coilRaw); raw != "" {
		coilRanges, err = parseAllowlist(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", envWriteAllowlistCoils, err)
		}
	}

	return &WritePolicy{
		enabled:      enabled,
		holdingRange: holdingRanges,
		coilsRange:   coilRanges,
	}, nil
}

func (wp *WritePolicy) Enabled() bool {
	if wp == nil {
		return false
	}
	return wp.enabled
}

func (wp *WritePolicy) ValidateHoldingWrite(address uint16, quantity int) error {
	return wp.validateWrite("holding-register", address, quantity, wp.holdingRange)
}

func (wp *WritePolicy) ValidateCoilWrite(address uint16, quantity int) error {
	return wp.validateWrite("coil", address, quantity, wp.coilsRange)
}

func (wp *WritePolicy) validateWrite(target string, address uint16, quantity int, allowlist []addressRange) error {
	if wp == nil || !wp.enabled {
		return fmt.Errorf("guarded rejection: write blocked by policy (%s=false)", envWritesEnabled)
	}
	if quantity <= 0 {
		return fmt.Errorf("guarded rejection: write quantity must be greater than 0")
	}

	if len(allowlist) == 0 {
		return nil
	}

	end := int(address) + quantity - 1
	if end > 65535 {
		return fmt.Errorf("guarded rejection: write range %d-%d exceeds Modbus address space", address, end)
	}

	if !isRangeAllowed(address, uint16(end), allowlist) {
		return fmt.Errorf("guarded rejection: %s write range %d-%d not in allowlist", target, address, end)
	}

	return nil
}

func parseAllowlist(raw string) ([]addressRange, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	ranges := make([]addressRange, 0, len(parts))
	for _, token := range parts {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		bounds := strings.Split(token, "-")
		switch len(bounds) {
		case 1:
			v, err := parseUint16(bounds[0])
			if err != nil {
				return nil, fmt.Errorf("invalid address %q", token)
			}
			ranges = append(ranges, addressRange{start: v, end: v})
		case 2:
			start, err := parseUint16(bounds[0])
			if err != nil {
				return nil, fmt.Errorf("invalid range start %q", token)
			}
			end, err := parseUint16(bounds[1])
			if err != nil {
				return nil, fmt.Errorf("invalid range end %q", token)
			}
			if start > end {
				return nil, fmt.Errorf("range start greater than end %q", token)
			}
			ranges = append(ranges, addressRange{start: start, end: end})
		default:
			return nil, fmt.Errorf("invalid allowlist token %q", token)
		}
	}

	return ranges, nil
}

func isRangeAllowed(start uint16, end uint16, allowlist []addressRange) bool {
	for _, r := range allowlist {
		if start >= r.start && end <= r.end {
			return true
		}
	}
	return false
}

func parseEnvBool(name string, fallback bool) bool {
	val := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseUint16(raw string) (uint16, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(v), nil
}
