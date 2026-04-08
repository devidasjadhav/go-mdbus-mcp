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

type ReadHoldingTypedArgs struct {
	Address   uint16   `json:"address" jsonschema:"Starting address to read from"`
	Quantity  *uint16  `json:"quantity,omitempty" jsonschema:"Optional register count (derived from data_type when omitted)"`
	DataType  string   `json:"data_type" jsonschema:"Target data type: uint16,int16,uint32,int32,float32,string"`
	ByteOrder *string  `json:"byte_order,omitempty" jsonschema:"Optional byte order: big or little"`
	WordOrder *string  `json:"word_order,omitempty" jsonschema:"Optional word order: msw or lsw (multi-word types)"`
	Scale     *float64 `json:"scale,omitempty" jsonschema:"Optional multiplier applied to decoded numeric value"`
	Offset    *float64 `json:"offset,omitempty" jsonschema:"Optional additive offset applied to decoded numeric value"`
	SlaveID   *uint8   `json:"slave_id,omitempty" jsonschema:"Optional Modbus Slave ID (defaults to 1)"`
}

type WriteHoldingTypedArgs struct {
	Address      uint16   `json:"address" jsonschema:"Starting address to write to"`
	Quantity     *uint16  `json:"quantity,omitempty" jsonschema:"Optional register count (derived from data_type when omitted)"`
	DataType     string   `json:"data_type" jsonschema:"Target data type: uint16,int16,uint32,int32,float32,string"`
	NumericValue *float64 `json:"numeric_value,omitempty" jsonschema:"Typed numeric value for numeric data_type"`
	StringValue  *string  `json:"string_value,omitempty" jsonschema:"Typed string value for data_type=string"`
	ByteOrder    *string  `json:"byte_order,omitempty" jsonschema:"Optional byte order: big or little"`
	WordOrder    *string  `json:"word_order,omitempty" jsonschema:"Optional word order: msw or lsw (multi-word types)"`
	Scale        *float64 `json:"scale,omitempty" jsonschema:"Optional multiplier applied before encoding numeric value"`
	Offset       *float64 `json:"offset,omitempty" jsonschema:"Optional additive offset removed before encoding numeric value"`
	SlaveID      *uint8   `json:"slave_id,omitempty" jsonschema:"Optional Modbus Slave ID (defaults to 1)"`
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
