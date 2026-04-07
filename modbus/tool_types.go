package modbus

// ReadArgs defines the input schema for reading modbus data.
type ReadArgs struct {
	Address  uint16 `json:"address" jsonschema:"Starting address to read from"`
	Quantity uint16 `json:"quantity" jsonschema:"Number of registers or coils to read"`
	SlaveID  *uint8 `json:"slave_id,omitempty" jsonschema:"Optional Modbus Slave ID (defaults to 1)"`
}

// WriteHoldingRegistersArgs defines the input schema for writing holding registers.
type WriteHoldingRegistersArgs struct {
	Address uint16   `json:"address" jsonschema:"Starting address to write to"`
	Values  []uint16 `json:"values" jsonschema:"Array of uint16 values to write"`
	SlaveID *uint8   `json:"slave_id,omitempty" jsonschema:"Optional Modbus Slave ID (defaults to 1)"`
}

// WriteCoilsArgs defines the input schema for writing coils.
type WriteCoilsArgs struct {
	Address uint16 `json:"address" jsonschema:"Starting address to write to"`
	Values  []bool `json:"values" jsonschema:"Array of boolean values to write"`
	SlaveID *uint8 `json:"slave_id,omitempty" jsonschema:"Optional Modbus Slave ID (defaults to 1)"`
}

type ReadTagArgs struct {
	Name    string `json:"name" jsonschema:"Configured tag name to read"`
	SlaveID *uint8 `json:"slave_id,omitempty" jsonschema:"Optional Modbus Slave ID override"`
}

type WriteTagArgs struct {
	Name          string   `json:"name" jsonschema:"Configured tag name to write"`
	HoldingValues []uint16 `json:"holding_values,omitempty" jsonschema:"Values for holding-register tags"`
	CoilValues    []bool   `json:"coil_values,omitempty" jsonschema:"Values for coil tags"`
	NumericValue  *float64 `json:"numeric_value,omitempty" jsonschema:"Typed numeric value for holding-register tag"`
	StringValue   *string  `json:"string_value,omitempty" jsonschema:"Typed string value for holding-register string tag"`
	BoolValue     *bool    `json:"bool_value,omitempty" jsonschema:"Typed bool value for single coil tag"`
	SlaveID       *uint8   `json:"slave_id,omitempty" jsonschema:"Optional Modbus Slave ID override"`
}
