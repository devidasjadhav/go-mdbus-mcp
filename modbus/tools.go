package modbus

import "github.com/modelcontextprotocol/go-sdk/mcp"

// RegisterTools registers all available modbus tools to the MCP server.
func RegisterTools(s *mcp.Server, driver Driver, writePolicy *WritePolicy, tagMap *TagMap) {
	registerDataTools(s, driver, writePolicy)
	registerStatusTools(s, driver)
	registerTagTools(s, driver, writePolicy, tagMap)
}
