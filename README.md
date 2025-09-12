# Modbus MCP Server

This is a simplified MCP (Model Context Protocol) server that provides Modbus TCP connectivity with a focus on reading holding registers.

## Features

- **Modbus TCP Client**: Connects to Modbus TCP servers with per-operation connections
- **MCP Tools**: Provides 4 tools for reading/writing holding registers and coils
- **Modular Architecture**: Well-organized code structure with separate packages
- **Automatic Connection Management**: Fresh connections for each operation prevent timeouts
- **HTTP Transport**: Uses streamable HTTP transport for MCP communication
- **Debug Logging**: Includes detailed logging for troubleshooting

## Project Structure

```
.
├── main.go              # Entry point and server setup
├── config/
│   └── config.go        # Configuration structures
├── modbus/
│   ├── client.go        # Modbus client implementation
│   └── tools.go         # MCP tool definitions
├── docs/                # 📚 Comprehensive documentation
│   ├── README.md        # Documentation overview
│   ├── architecture.md  # System architecture & design
│   ├── api-reference.md # Complete API documentation
│   ├── development.md   # Development guide & workflow
│   └── deployment.md    # Production deployment guide
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── test_gemini.py       # Gemini integration test script
├── gemini_integration_guide.md # Complete Gemini integration guide
└── README.md            # This file (main project README)
```

## Prerequisites

- A running Modbus TCP server (for testing, you can use a simulator)
- Port 8080 available for the MCP server

## Usage

### Command Line Arguments

- `--modbus-ip`: Modbus server IP address (default: 192.168.1.22)
- `--modbus-port`: Modbus server port (default: 5002)

### Start the Server

```bash
# Connect to test Modbus server on default port
go run main.go

# Connect to specific Modbus server
go run main.go --modbus-ip 192.168.1.100 --modbus-port 502
```

### Available Tools

1. **read-holding-registers**: Read Modbus holding registers with automatic reconnection
2. **read-coils**: Read Modbus coils (digital inputs/outputs) with bit processing
3. **write-holding-registers**: Write values to Modbus holding registers
4. **write-coils**: Write values to Modbus coils (digital outputs)

### Example API Calls

#### List Available Tools

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"method":"tools/list", "id":1}' \
  http://localhost:8080/mcp
```

#### Read Holding Registers

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read-holding-registers","arguments":{"address":0,"quantity":10}},"id":1}' \
  http://localhost:8080/mcp
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "text": "Holding registers at address 0: [100 42 0 0 0 0 0 0 0 123]",
        "type": "text"
      }
    ]
  },
  "id": 1
}
```

#### Read Coils

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read-coils","arguments":{"address":0,"quantity":14}},"id":1}' \
  http://localhost:8080/mcp
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "text": "Coils at address 0: [false true true true true true false false false false false true false false]",
        "type": "text"
      }
    ]
  },
  "id": 1
}
```

#### Write Holding Registers

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"write-holding-registers","arguments":{"address":10,"values":[1234,5678,9999]}},"id":1}' \
  http://localhost:8080/mcp
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "text": "Successfully wrote 3 values to holding registers starting at address 10: [1234 5678 9999]",
        "type": "text"
      }
    ]
  },
  "id": 1
}
```

#### Write Coils

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"write-coils","arguments":{"address":5,"values":[true,false,true,true,false]}},"id":1}' \
  http://localhost:8080/mcp
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "text": "Successfully wrote 5 values to coils starting at address 5: [true false true true false]",
        "type": "text"
      }
    ]
  },
  "id": 1
}
```

#### Read Coils

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read-coils","arguments":{"address":0,"quantity":8}},"id":2}' \
  http://localhost:8080/mcp
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "text": "Coils at address 0: [true false true false false false false true]",
        "type": "text"
      }
    ]
  },
  "id": 2
}
```

## Connection Management

The server uses a per-operation connection strategy to prevent timeout issues:

- **Per-Operation Connections**: Creates a fresh TCP connection for each read operation and closes it immediately after
- **No Persistent Connections**: Eliminates connection timeout problems that occur with idle persistent connections
- **Automatic Cleanup**: Connections are automatically closed using defer statements to ensure proper resource management
- **Fresh Handler Creation**: Each operation creates a new Modbus handler for reliable communication
- **Timeout Handling**: 10-second timeout per operation for reliable operation with slower networks
- **Debug Logging**: Detailed logs help troubleshoot connection and communication issues

## Testing

### Automated Tests

Run the automated tests:

```bash
go test ./...
```

Or with mcp-autotest:

```bash
mcp-autotest run -u http://localhost:8080/mcp testdata -- go run main.go --modbus-ip 127.0.0.1 --modbus-port 502
```

### Manual Testing

Build and run the server:

```bash
go build -o modbus-server main.go
./modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002
```

## 📚 Documentation

For comprehensive documentation, see the [`docs/`](./docs/) directory:

- **[📋 Documentation Overview](./docs/README.md)** - Complete guide to all documentation
- **[🏗️ Architecture](./docs/architecture.md)** - System design, data flow, and design decisions
- **[🔧 API Reference](./docs/api-reference.md)** - Complete API documentation with examples
- **[💻 Development Guide](./docs/development.md)** - Setup, workflow, and contribution guidelines
- **[🚀 Deployment Guide](./docs/deployment.md)** - Production deployment and configuration

**Quick Start for Documentation:**
1. **New to the project?** → Start with [Architecture](./docs/architecture.md)
2. **Need API details?** → Check [API Reference](./docs/api-reference.md)
3. **Want to contribute?** → Read [Development Guide](./docs/development.md)
4. **Deploying to production?** → Follow [Deployment Guide](./docs/deployment.md)

### Manual Testing with mcptools

The server has been tested and verified to work correctly with the MCP protocol. You can test it using the [mcptools](https://github.com/f/mcptools) CLI tool.

#### Installation

```bash
# Install mcptools
go install github.com/f/mcptools/cmd/mcptools@latest

# Make sure it's in your PATH or use the full path
~/go/bin/mcptools --help
```

#### Testing the Server

1. **Start the MCP server:**
   ```bash
   cd sample
   go run main.go
   ```

2. **Test server functionality with direct HTTP requests:**
   ```bash
   # Initialize connection
   curl -X POST -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}},"id":0}' \
     http://localhost:8080/mcp

   # List available tools
   curl -X POST -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' \
     http://localhost:8080/mcp

   # Call a tool
   curl -X POST -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"my-great-tool","arguments":{}},"id":2}' \
     http://localhost:8080/mcp
   ```

#### Expected Results

✅ **Server Status**: The server starts successfully on `http://localhost:8080/mcp`

✅ **Available Tools** (2 tools):
- `read-holding-registers` - Read Modbus holding registers with per-operation connections
- `read-coils` - Read Modbus coils (digital outputs) with per-operation connections

✅ **Tool Functionality**:
- **Read Operations**:
  - Returns holding register values as uint16 arrays: `[100 42 0 0 0 0 0 0 0 123]`
  - Returns coil states as boolean arrays: `[false true true true true true false false]`
  - Properly processes Modbus bit-packed coil data (8 coils per byte)
- **Write Operations**:
  - Writes multiple holding register values: `[1234 5678 9999]`
  - Writes multiple coil states: `[true false true true false]`
  - Automatic data conversion (uint16[] to bytes for registers, bool[] to bit-packed bytes for coils)
- Uses per-operation connections to prevent timeout issues
- Provides detailed error messages for connection issues

✅ **MCP Protocol Compliance**: Server properly handles initialization, tool listing, and tool calling according to MCP specification

✅ **Connection Resilience**: Server automatically reconnects when the Modbus server connection is lost

## Modbus Configuration

- **Slave ID**: Fixed to 0 (common default for Modbus TCP servers)
- **Timeout**: 10 seconds (increased for better reliability)
- **Connection**: TCP with per-operation connections (connect → read → close)
- **Connection Management**: Fresh connections for each operation to prevent timeouts

## Recent Improvements

### Version Updates
- **Modular Architecture**: Refactored code into organized packages (config, modbus)
- **Enhanced Tool Set**: Added write functionality for both holding registers and coils
- **Per-Operation Connections**: Fixed timeout issues by using fresh connections for each operation
- **Improved Code Organization**: Separated concerns into logical modules
- **Automatic Connection Cleanup**: Proper resource management with automatic connection closing
- **Debug Logging**: Added detailed logging for troubleshooting connection and communication issues

### Connection Resilience
The server now handles network interruptions gracefully:
- Uses per-operation connections to prevent timeout issues
- Creates fresh Modbus handlers for each operation
- Automatic connection cleanup prevents resource leaks
- Provides clear error messages for debugging
- Eliminates persistent connection timeout problems

## Error Handling

The server provides detailed error messages and automatic recovery for:
- **Modbus connection failures**: Automatic reconnection attempts
- **Invalid addresses or quantities**: Clear validation error messages
- **Communication timeouts**: 10-second timeout with retry logic
- **Protocol errors**: Detailed logging for troubleshooting
- **Connection drops**: Graceful handling with handler recreation