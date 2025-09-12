# Architecture Documentation

## Overview

The Modbus MCP Server is a Model Context Protocol (MCP) server that provides Modbus TCP connectivity for industrial automation systems. It enables AI assistants and other MCP clients to read from and write to Modbus devices through a standardized interface.

## System Architecture

### High-Level Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   MCP Client    │────│  Modbus MCP      │────│  Modbus Device  │
│  (AI Assistant) │    │     Server       │    │   (PLC, etc.)   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │  Configuration   │
                       │    & Logging     │
                       └──────────────────┘
```

### Component Architecture

```
Modbus MCP Server
├── main.go (Entry Point)
├── config/
│   └── config.go (Configuration)
├── modbus/
│   ├── client.go (Modbus Client)
│   └── tools.go (MCP Tools)
└── docs/ (Documentation)
```

## Component Details

### 1. Main Entry Point (`main.go`)

**Responsibilities:**
- Parse command-line arguments
- Initialize configuration
- Create Modbus client instance
- Register MCP tools with the server
- Start HTTP server with MCP transport

**Key Features:**
- Command-line argument parsing (`--modbus-ip`, `--modbus-port`)
- Server lifecycle management
- Error handling and graceful shutdown

### 2. Configuration (`config/config.go`)

**Structure:**
```go
type Config struct {
    ModbusIP   string
    ModbusPort int
}
```

**Responsibilities:**
- Define configuration structure
- Centralize configuration management
- Support for different environments

### 3. Modbus Client (`modbus/client.go`)

**Key Components:**
- `ModbusClient` struct
- Connection management methods
- Error handling and reconnection logic

**Connection Strategy:**
- **Per-operation connections**: Creates fresh TCP connection for each operation
- **Automatic cleanup**: Closes connections immediately after use
- **Timeout prevention**: Eliminates persistent connection timeout issues

**Methods:**
- `NewModbusClient()`: Factory function for client creation
- `Connect()`: Establish TCP connection
- `Close()`: Close TCP connection
- `EnsureConnected()`: Ensure connection is ready for operation

### 4. MCP Tools (`modbus/tools.go`)

**Available Tools:**
1. `read-holding-registers` - Read uint16 values from holding registers
2. `read-coils` - Read boolean values from coils
3. `write-holding-registers` - Write uint16 values to holding registers
4. `write-coils` - Write boolean values to coils

**Tool Structure:**
Each tool implements the MCP `Tool` interface with:
- **Name**: Unique identifier
- **Description**: Human-readable description
- **InputSchema**: JSON schema for parameters
- **Handler**: Function to process requests

## Data Flow

### Read Operation Flow

```
1. MCP Client Request
       ↓
2. Tool Handler (tools.go)
       ↓
3. Ensure Connection (client.go)
       ↓
4. Modbus Read Operation
       ↓
5. Data Processing & Response
       ↓
6. Connection Cleanup
       ↓
7. MCP Response to Client
```

### Write Operation Flow

```
1. MCP Client Request
       ↓
2. Tool Handler (tools.go)
       ↓
3. Parameter Validation
       ↓
4. Data Conversion (uint16/bool → bytes)
       ↓
5. Ensure Connection (client.go)
       ↓
6. Modbus Write Operation
       ↓
7. Connection Cleanup
       ↓
8. Success Response
```

## Design Decisions

### 1. Per-Operation Connections

**Decision**: Use fresh TCP connections for each operation instead of persistent connections.

**Rationale:**
- Prevents connection timeout issues with idle persistent connections
- Simplifies connection management
- Reduces resource usage for infrequent operations
- Improves reliability in network environments with firewalls

**Trade-offs:**
- Slightly higher latency due to connection establishment
- More network overhead for frequent operations
- Simpler error handling and recovery

### 2. Modular Architecture

**Decision**: Organize code into separate packages (`config`, `modbus`).

**Benefits:**
- Clear separation of concerns
- Improved maintainability
- Easier testing
- Better code organization
- Scalability for future features

### 3. MCP Protocol Compliance

**Decision**: Strictly follow MCP specification for tool definitions and communication.

**Benefits:**
- Interoperability with any MCP client
- Standardized interface
- Future-proof design
- Community ecosystem compatibility

### 4. Error Handling Strategy

**Decision**: Comprehensive error handling with detailed error messages.

**Implementation:**
- Parameter validation at tool level
- Connection error handling
- Modbus protocol error handling
- Clear error messages for debugging

## Communication Protocols

### 1. MCP (Model Context Protocol)

**Transport**: HTTP with JSON-RPC 2.0
**Port**: 8080 (configurable)
**Endpoint**: `/mcp`

**Message Format:**
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

### 2. Modbus TCP

**Protocol**: Modbus TCP/IP
**Port**: 502 (default, configurable)
**Function Codes**:
- 0x03: Read Holding Registers
- 0x01: Read Coils
- 0x10: Write Multiple Registers
- 0x0F: Write Multiple Coils

## Security Considerations

### 1. Network Security
- No authentication/authorization implemented
- Consider adding TLS for production use
- Network segmentation recommended

### 2. Input Validation
- Parameter type checking
- Range validation for addresses and quantities
- Sanitization of all inputs

### 3. Error Information
- Avoid exposing internal system details in error messages
- Log detailed errors internally while providing user-friendly messages

## Performance Characteristics

### Connection Management
- **Connection Time**: ~10-50ms per operation
- **Memory Usage**: Minimal (connections cleaned up immediately)
- **Concurrent Operations**: Supported (each operation independent)

### Modbus Operations
- **Read Latency**: Depends on network and device response time
- **Write Latency**: Depends on network and device processing time
- **Batch Operations**: Supported for multiple registers/coils

## Monitoring and Observability

### Logging
- **Connection Events**: Connection establishment and cleanup
- **Modbus Operations**: Read/write operations with parameters
- **Errors**: Detailed error logging for troubleshooting
- **Performance**: Operation timing and success/failure metrics

### Health Checks
- **Connection Status**: Ability to verify Modbus connectivity
- **Tool Availability**: All tools should be responsive
- **Error Rates**: Monitor for increasing error patterns

## Future Enhancements

### Potential Architecture Improvements
1. **Connection Pooling**: For high-frequency operations
2. **Caching**: For frequently read values
3. **Batch Operations**: Optimize multiple operations
4. **Authentication**: Add security layers
5. **Metrics**: Add Prometheus metrics endpoint
6. **Configuration**: External configuration files
7. **TLS Support**: Encrypted communication

### Scalability Considerations
1. **Horizontal Scaling**: Multiple server instances
2. **Load Balancing**: Distribute requests across instances
3. **Database Integration**: For configuration and logging
4. **Message Queues**: For asynchronous operations

## Dependencies

### Core Dependencies
- **github.com/goburrow/modbus**: Modbus protocol implementation
- **github.com/strowk/foxy-contexts**: MCP framework
- **go.uber.org/fx**: Dependency injection framework
- **go.uber.org/zap**: Logging framework

### Development Dependencies
- **github.com/goburrow/serial**: Serial communication (indirect)
- **golang.org/x/net**: Network utilities (indirect)

## Testing Strategy

### Unit Tests
- Individual component testing
- Mock Modbus server for testing
- Tool handler validation
- Error condition testing

### Integration Tests
- End-to-end MCP communication
- Real Modbus device testing
- Performance testing
- Load testing

### Test Coverage
- Target: >80% code coverage
- Focus on critical paths
- Error handling scenarios
- Edge cases and boundary conditions