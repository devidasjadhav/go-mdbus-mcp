package modbus

import "github.com/modelcontextprotocol/go-sdk/mcp"

// RegisterTools registers all available modbus tools to the MCP server.
func RegisterTools(s *mcp.Server, mc *ModbusClient, writePolicy *WritePolicy, tagMap *TagMap) {
	registerDataTools(s, mc, writePolicy)
	registerStatusTools(s, mc)
	registerTagTools(s, mc, writePolicy, tagMap)
}
