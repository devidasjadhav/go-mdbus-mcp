# Tag CSV Reference

## Required columns

- `name`
- `kind` (`holding_register` or `coil`)
- `address`

## Optional columns

- `quantity`
- `slave_id`
- `access` (`read`, `write`, `read_write`)
- `data_type`
- `byte_order` (`big`, `little`)
- `word_order` (`msw`, `lsw`)
- `scale`
- `offset`
- `description`

## Supported data types

Holding-register tags:

- `uint16`, `int16`
- `uint32`, `int32`
- `float32`
- `string`

Coil tags:

- `bool`

## Quantity rules

- `uint16`, `int16` default to quantity `1`
- `uint32`, `int32`, `float32` default to quantity `2`
- `string` requires explicit quantity
- `coil` defaults to quantity `1`

## Example

```csv
name,kind,address,quantity,slave_id,access,data_type,byte_order,word_order,scale,offset,description
ambient_temp_raw,holding_register,0,1,1,read,uint16,big,msw,1,0,Raw ambient temperature register
boiler_temp_c,holding_register,10,2,1,read,float32,big,msw,1,0,Boiler temperature in Celsius
device_label,holding_register,20,4,1,read,string,big,msw,1,0,ASCII device label
run_command,coil,0,1,1,read_write,bool,big,msw,1,0,Run command coil
```

## Typed write-tag inputs

- `numeric_value`: for numeric holding-register tags
- `string_value`: for holding-register string tags
- `bool_value`: for single-coil tags (`quantity=1`)

Raw arrays are also supported:

- `holding_values`
- `coil_values`
