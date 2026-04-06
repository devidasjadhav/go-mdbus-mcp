# Modbus MCP Server

A lightweight MCP (Model Context Protocol) server for Modbus TCP connectivity.

## Features

- **Modbus TCP Client**: Connects to Modbus TCP servers with per-operation connections.
- **MCP Tools**: Provides tools for reading and writing Modbus holding registers and coils.
- **HTTP Transport**: Uses streamable HTTP transport for MCP communication.

## Quick Start

### Build & Run

```bash
go build -o modbus-server main.go

# Run with default settings (Connects to 192.168.1.22:5002)
./modbus-server

# Run with specific IP/Port
./modbus-server --modbus-ip 192.168.1.100 --modbus-port 502
```

### Docker

```bash
docker build -t modbus-mcp-server .
docker run -p 8080:8080 -p 8081:8081 modbus-mcp-server
```

## Available Tools

1. `read-holding-registers`: Read Modbus holding registers (returns `uint16` values).
2. `read-coils`: Read Modbus coils (returns boolean values).
3. `write-holding-registers`: Write values to Modbus holding registers.
4. `write-coils`: Write values to Modbus coils.

## Testing

```bash
go test ./...
```
