# Modbus MCP Server

This is a simplified MCP (Model Context Protocol) server that provides Modbus TCP connectivity with a focus on reading holding registers.

## Features

- **Modbus TCP Client**: Connects to Modbus TCP servers with automatic reconnection
- **MCP Tools**: Provides a single tool for reading holding registers
- **Automatic Reconnection**: Handles connection drops gracefully by recreating handlers
- **HTTP Transport**: Uses streamable HTTP transport for MCP communication
- **Debug Logging**: Includes detailed logging for troubleshooting

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

## Connection Management

The server includes robust connection management:

- **Automatic Reconnection**: If the connection to the Modbus server drops, the server will automatically attempt to reconnect
- **Handler Recreation**: When reconnection fails, the server recreates the Modbus handler to ensure clean connections
- **Timeout Handling**: 10-second timeout for reliable operation with slower networks
- **Debug Logging**: Detailed logs help troubleshoot connection issues

## Testing

### Automated Tests

Run the automated tests:

```bash
go test
```

Or with mcp-autotest:

```bash
mcp-autotest run -u http://localhost:8080/mcp testdata -- go run main.go --modbus-ip 127.0.0.1 --modbus-port 502
```

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

✅ **Available Tools** (1 tool):
- `read-holding-registers` - Read Modbus holding registers with automatic reconnection

✅ **Tool Functionality**:
- Returns holding register values as uint16 arrays: `[100 42 0 0 0 0 0 0 0 123]`
- Handles connection drops gracefully with automatic reconnection
- Provides detailed error messages for connection issues

✅ **MCP Protocol Compliance**: Server properly handles initialization, tool listing, and tool calling according to MCP specification

✅ **Connection Resilience**: Server automatically reconnects when the Modbus server connection is lost

## Modbus Configuration

- **Slave ID**: Fixed to 0 (common default for Modbus TCP servers)
- **Timeout**: 10 seconds (increased for better reliability)
- **Connection**: TCP with automatic reconnection and handler recreation
- **Reconnection**: Automatic with exponential backoff and handler recreation

## Recent Improvements

### Version Updates
- **Simplified Architecture**: Focused on read-holding-registers functionality only
- **Enhanced Reliability**: Increased timeout to 10 seconds for better network handling
- **Improved Slave ID**: Changed to 0 (standard default for Modbus TCP)
- **Automatic Reconnection**: Robust connection recovery with handler recreation
- **Debug Logging**: Added detailed logging for troubleshooting connection issues

### Connection Resilience
The server now handles network interruptions gracefully:
- Detects connection drops automatically
- Recreates Modbus handlers when needed
- Retries connections with proper error handling
- Provides clear error messages for debugging

## Error Handling

The server provides detailed error messages and automatic recovery for:
- **Modbus connection failures**: Automatic reconnection attempts
- **Invalid addresses or quantities**: Clear validation error messages
- **Communication timeouts**: 10-second timeout with retry logic
- **Protocol errors**: Detailed logging for troubleshooting
- **Connection drops**: Graceful handling with handler recreation