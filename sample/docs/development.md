# Development Guide

This guide provides information for developers who want to contribute to or modify the Modbus MCP Server.

## Prerequisites

### System Requirements
- **Go**: Version 1.21 or later
- **Git**: For version control
- **Modbus Device/Server**: For testing (optional, can use simulator)

### Development Tools
- **Go toolchain**: `go build`, `go test`, `go mod`
- **Git**: Version control
- **curl**: API testing
- **Docker**: Optional for containerized development

## Getting Started

### 1. Clone the Repository
```bash
git clone https://github.com/devidasjadhav/go-mdbus-mcp.git
cd go-mdbus-mcp/sample
```

### 2. Install Dependencies
```bash
go mod download
```

### 3. Build the Project
```bash
go build -o modbus-server main.go
```

### 4. Run the Server
```bash
# With default settings (connects to 192.168.1.22:5002)
./modbus-server

# With custom Modbus server
./modbus-server --modbus-ip 127.0.0.1 --modbus-port 502
```

### 5. Test the Server
```bash
# List available tools
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' \
  http://localhost:8080/mcp

# Test a read operation
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read-holding-registers","arguments":{"address":0,"quantity":5}},"id":2}' \
  http://localhost:8080/mcp
```

## Project Structure

```
sample/
├── main.go              # Application entry point
├── config/
│   └── config.go        # Configuration structures
├── modbus/
│   ├── client.go        # Modbus client implementation
│   └── tools.go         # MCP tool definitions
├── docs/                # Documentation
│   ├── README.md        # Documentation overview
│   ├── architecture.md  # System architecture
│   ├── api-reference.md # API documentation
│   ├── development.md   # This file
│   └── deployment.md    # Deployment guide
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
└── README.md            # Main project README
```

## Development Workflow

### 1. Create a Feature Branch
```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes
- Follow the existing code style and patterns
- Add tests for new functionality
- Update documentation as needed
- Ensure code compiles without warnings

### 3. Test Your Changes
```bash
# Run tests
go test ./...

# Build the project
go build -o modbus-server main.go

# Test manually with curl or MCP client
```

### 4. Commit Your Changes
```bash
git add .
git commit -m "Add feature: brief description of changes"
```

### 5. Push and Create Pull Request
```bash
git push origin feature/your-feature-name
# Create pull request on GitHub
```

## Code Organization

### Package Structure

#### `main.go`
- Entry point for the application
- Command-line argument parsing
- Server initialization and startup
- Dependency injection setup

#### `config/`
- Configuration structures and validation
- Environment-specific settings
- Default values and constants

#### `modbus/`
- **client.go**: Modbus TCP client implementation
  - Connection management
  - Error handling and reconnection
  - Modbus protocol operations

- **tools.go**: MCP tool definitions
  - Tool registration and schemas
  - Request/response handling
  - Parameter validation

### Coding Standards

#### Go Conventions
- Follow standard Go naming conventions
- Use `gofmt` for code formatting
- Write clear, concise function names
- Add comments for exported functions

#### Error Handling
- Return errors rather than panicking
- Provide meaningful error messages
- Log errors with appropriate context
- Handle edge cases gracefully

#### Logging
- Use structured logging with context
- Log important operations and errors
- Include relevant parameters in log messages
- Use appropriate log levels

### Adding New Tools

#### 1. Define the Tool Schema
```go
// In modbus/tools.go
func NewYourTool(mc *ModbusClient) fxctx.Tool {
    return fxctx.NewTool(
        &mcp.Tool{
            Name:        "your-tool-name",
            Description: Ptr("Description of what the tool does"),
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]map[string]interface{}{
                    "parameter1": {
                        "type":        "string",
                        "description": "Description of parameter1",
                    },
                    // Add more parameters as needed
                },
                Required: []string{"parameter1"},
            },
        },
        // Tool handler function
        func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
            // Implementation here
        },
    )
}
```

#### 2. Register the Tool
```go
// In main.go
server := app.
    NewBuilder().
    WithTool(func() fxctx.Tool { return modbus.NewYourTool(modbusClient) }).
    // ... other tools
```

#### 3. Add Tests
```go
// In your test file
func TestYourTool(t *testing.T) {
    // Test implementation
}
```

## Testing

### Unit Tests
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./modbus/...

# Run tests with verbose output
go test -v ./...
```

### Integration Tests
```bash
# Test with real Modbus server
go run main.go --modbus-ip 127.0.0.1 --modbus-port 502

# Use testing tools
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' \
  http://localhost:8080/mcp
```

### Test Coverage
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# View coverage in browser
open coverage.html
```

## Debugging

### Enable Debug Logging
The server provides detailed logging for troubleshooting:

```bash
# Server logs include:
# - Connection establishment
# - Modbus operations with parameters
# - Error details and stack traces
# - Performance timing
```

### Common Debug Scenarios

#### Connection Issues
```bash
# Check if Modbus server is reachable
telnet 192.168.1.22 5002

# Test with different IP/port
./modbus-server --modbus-ip 127.0.0.1 --modbus-port 502
```

#### Tool Errors
```bash
# Test individual tools
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read-coils","arguments":{"address":0,"quantity":1}},"id":1}' \
  http://localhost:8080/mcp
```

#### Performance Issues
```bash
# Monitor server logs for timing information
# Check network latency to Modbus server
# Verify Modbus server performance
```

## Performance Optimization

### Connection Management
- Per-operation connections prevent timeout issues
- Automatic connection cleanup reduces memory usage
- Concurrent operations supported

### Modbus Operations
- Batch read/write operations for efficiency
- Optimal parameter validation
- Error handling without performance impact

### Memory Management
- No persistent connections means lower memory footprint
- Automatic garbage collection of temporary objects
- Efficient data structures for Modbus operations

## Security Considerations

### Input Validation
- All parameters are validated before processing
- Type checking for all inputs
- Range validation for addresses and quantities

### Error Handling
- No sensitive information exposed in error messages
- Internal errors logged but not returned to clients
- Graceful handling of malformed requests

### Network Security
- No authentication implemented (add as needed)
- Consider TLS for production deployments
- Network segmentation recommended

## Contributing Guidelines

### Pull Request Process
1. **Fork** the repository
2. **Create** a feature branch
3. **Make** your changes
4. **Add tests** for new functionality
5. **Update documentation** as needed
6. **Ensure** all tests pass
7. **Submit** a pull request

### Code Review Checklist
- [ ] Code follows Go conventions
- [ ] Tests added for new functionality
- [ ] Documentation updated
- [ ] No breaking changes without discussion
- [ ] Performance impact considered
- [ ] Security implications reviewed

### Commit Message Format
```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Testing
- `chore`: Maintenance

## Troubleshooting Development Issues

### Build Issues
```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Check for Go version compatibility
go version
```

### Test Issues
```bash
# Run tests with race detection
go test -race ./...

# Run tests with verbose output
go test -v ./...

# Debug failing tests
go test -run TestSpecificFunction ./...
```

### IDE Setup
- Use Go extension for VS Code or similar
- Enable gofmt on save
- Configure Go tools (goimports, golint, etc.)
- Set up debugging for the main.go file

## Advanced Development

### Adding New Modbus Functions
1. Research the Modbus function code
2. Implement the function in the client
3. Create MCP tool wrapper
4. Add parameter validation
5. Test with real hardware

### Custom Data Types
1. Define the data structure
2. Implement encoding/decoding logic
3. Add to tool schema
4. Update documentation

### Performance Monitoring
1. Add metrics collection
2. Implement health checks
3. Monitor connection pool usage
4. Track operation latency

## Resources

### Go Documentation
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### Modbus Resources
- [Modbus Protocol Specification](https://modbus.org/docs/Modbus_Application_Protocol_V1_1b3.pdf)
- [Modbus TCP/IP Implementation Guide](https://modbus.org/docs/Modbus_Messaging_Implementation_Guide_V1_0b.pdf)

### MCP Resources
- [Model Context Protocol Specification](https://modelcontextprotocol.io/specification)
- [MCP SDK Documentation](https://modelcontextprotocol.io/sdk)

## Support

For development questions:
1. Check existing issues and documentation
2. Review the codebase for similar implementations
3. Create an issue for bugs or feature requests
4. Join community discussions

---

*This development guide is maintained alongside the codebase. Please update it when making significant changes to the development process.*