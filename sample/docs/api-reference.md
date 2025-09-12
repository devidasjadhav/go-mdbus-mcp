# API Reference

This document provides detailed API reference for the Modbus MCP Server tools and their usage.

## Overview

The Modbus MCP Server provides 4 MCP tools for interacting with Modbus devices:

1. **read-holding-registers** - Read 16-bit integer values from holding registers
2. **read-coils** - Read boolean values from coils (digital inputs/outputs)
3. **write-holding-registers** - Write 16-bit integer values to holding registers
4. **write-coils** - Write boolean values to coils (digital outputs)

## Tool Specifications

### 1. read-holding-registers

**Description**: Read one or more 16-bit integer values from Modbus holding registers.

**Parameters:**
```json
{
  "address": {
    "type": "integer",
    "description": "Starting address to read from (0-65535)",
    "minimum": 0,
    "maximum": 65535
  },
  "quantity": {
    "type": "integer",
    "description": "Number of registers to read (1-125)",
    "minimum": 1,
    "maximum": 125
  }
}
```

**Required Parameters**: `address`, `quantity`

**Example Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "read-holding-registers",
    "arguments": {
      "address": 0,
      "quantity": 10
    }
  },
  "id": 1
}
```

**Example Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Holding registers at address 0: [1000 420 0 0 0 0 0 0 0 123]"
      }
    ]
  },
  "id": 1
}
```

**Error Responses:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Error reading holding registers: connection timeout"
      }
    ],
    "isError": true
  },
  "id": 1
}
```

### 2. read-coils

**Description**: Read one or more boolean values from Modbus coils.

**Parameters:**
```json
{
  "address": {
    "type": "integer",
    "description": "Starting address to read from (0-65535)",
    "minimum": 0,
    "maximum": 65535
  },
  "quantity": {
    "type": "integer",
    "description": "Number of coils to read (1-2000)",
    "minimum": 1,
    "maximum": 2000
  }
}
```

**Required Parameters**: `address`, `quantity`

**Example Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "read-coils",
    "arguments": {
      "address": 0,
      "quantity": 8
    }
  },
  "id": 1
}
```

**Example Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Coils at address 0: [false true true true true true false true]"
      }
    ]
  },
  "id": 1
}
```

### 3. write-holding-registers

**Description**: Write one or more 16-bit integer values to Modbus holding registers.

**Parameters:**
```json
{
  "address": {
    "type": "integer",
    "description": "Starting address to write to (0-65535)",
    "minimum": 0,
    "maximum": 65535
  },
  "values": {
    "type": "array",
    "description": "Array of uint16 values to write",
    "items": {
      "type": "integer",
      "minimum": 0,
      "maximum": 65535
    },
    "minItems": 1,
    "maxItems": 123
  }
}
```

**Required Parameters**: `address`, `values`

**Example Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "write-holding-registers",
    "arguments": {
      "address": 10,
      "values": [1234, 5678, 9999]
    }
  },
  "id": 1
}
```

**Example Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Successfully wrote 3 values to holding registers starting at address 10: [1234 5678 9999]"
      }
    ]
  },
  "id": 1
}
```

### 4. write-coils

**Description**: Write one or more boolean values to Modbus coils.

**Parameters:**
```json
{
  "address": {
    "type": "integer",
    "description": "Starting address to write to (0-65535)",
    "minimum": 0,
    "maximum": 65535
  },
  "values": {
    "type": "array",
    "description": "Array of boolean values to write",
    "items": {
      "type": "boolean"
    },
    "minItems": 1,
    "maxItems": 1968
  }
}
```

**Required Parameters**: `address`, `values`

**Example Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "write-coils",
    "arguments": {
      "address": 5,
      "values": [true, false, true, true, false]
    }
  },
  "id": 1
}
```

**Example Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Successfully wrote 5 values to coils starting at address 5: [true false true true false]"
      }
    ]
  },
  "id": 1
}
```

## Common Error Codes

### Connection Errors
- **"Failed to connect to Modbus server"**: Cannot establish TCP connection
- **"connection timeout"**: Modbus server not responding
- **"write tcp: broken pipe"**: Connection was unexpectedly closed

### Parameter Errors
- **"Invalid address parameter"**: Address not a valid number
- **"Invalid quantity parameter"**: Quantity not a valid number or out of range
- **"Invalid values parameter"**: Values array malformed or contains invalid data

### Modbus Protocol Errors
- **"Illegal function"**: Unsupported Modbus function code
- **"Illegal data address"**: Invalid register/coil address
- **"Illegal data value"**: Invalid data value for the operation
- **"Slave device failure"**: Modbus device reported an error

## Data Types and Ranges

### Address Ranges
- **Holding Registers**: 0-65535 (16-bit address space)
- **Coils**: 0-65535 (16-bit address space)

### Quantity Limits
- **Read Holding Registers**: 1-125 registers per request
- **Read Coils**: 1-2000 coils per request
- **Write Holding Registers**: 1-123 registers per request
- **Write Coils**: 1-1968 coils per request

### Data Value Ranges
- **Holding Registers**: 0-65535 (16-bit unsigned integer)
- **Coils**: true/false (boolean)

## Protocol Details

### Modbus Function Codes Used
- **0x03 (3)**: Read Holding Registers
- **0x01 (1)**: Read Coils
- **0x10 (16)**: Write Multiple Registers
- **0x0F (15)**: Write Multiple Coils

### Data Encoding
- **Holding Registers**: Big-endian 16-bit integers
- **Coils**: Bit-packed into bytes (8 coils per byte)

### Connection Management
- **Per-operation connections**: Fresh TCP connection for each request
- **Automatic cleanup**: Connections closed immediately after operation
- **Timeout**: 10 seconds per operation
- **Slave ID**: Fixed to 0 (standard for Modbus TCP)

## Rate Limiting and Performance

### Operation Timing
- **Connection establishment**: ~10-50ms
- **Modbus operation**: Depends on network and device
- **Total per operation**: ~50-200ms (typical)

### Concurrent Operations
- **Supported**: Multiple operations can run concurrently
- **Independent**: Each operation uses its own connection
- **Resource usage**: Minimal due to connection cleanup

### Best Practices
1. **Batch operations**: Use multiple values in single request when possible
2. **Sequential access**: Read/write consecutive addresses for efficiency
3. **Error handling**: Implement retry logic for transient errors
4. **Monitoring**: Track operation success/failure rates

## Troubleshooting

### Common Issues

#### Connection Problems
**Symptom**: "Failed to connect to Modbus server"
**Solutions**:
- Verify Modbus server IP address and port
- Check network connectivity
- Ensure Modbus server is running
- Verify firewall settings

#### Timeout Issues
**Symptom**: "connection timeout" or "EOF"
**Solutions**:
- Increase timeout value if needed
- Check network latency
- Verify Modbus server responsiveness
- Consider network congestion

#### Invalid Responses
**Symptom**: Unexpected data or errors
**Solutions**:
- Verify address ranges and data types
- Check Modbus device configuration
- Validate parameter values
- Review Modbus device documentation

### Debug Information

Enable debug logging to troubleshoot issues:
```bash
# Server logs connection and operation details
# Check server output for detailed error information
```

### Testing Tools

Use these tools to test Modbus connectivity:
- **Modbus Poll**: Windows Modbus testing tool
- **Modbus Simulator**: Test server for development
- **Wireshark**: Network packet analysis
- **curl**: Direct API testing

## Version Information

- **API Version**: MCP 2024-11-05
- **Modbus Protocol**: Modbus TCP/IP
- **Transport**: HTTP with JSON-RPC 2.0
- **Default Port**: 8080 (MCP), 502 (Modbus)

## Changelog

### Version 0.0.1
- Initial release with 4 MCP tools
- Per-operation connection strategy
- Comprehensive error handling
- Modular architecture