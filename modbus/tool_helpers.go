package modbus

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// executeTool is a helper to run thread-safe operations on the Modbus client
// and handle any protocol errors.
func executeTool(ctx context.Context, driver Driver, slaveID *uint8, allowRetry bool, operation func() (*mcp.CallToolResult, error)) (*mcp.CallToolResult, any, error) {
	targetSlaveID := uint8(1)
	if slaveID != nil {
		targetSlaveID = *slaveID
	}

	res, err := driver.Execute(ctx, targetSlaveID, allowRetry, operation)
	if err != nil {
		// As per the official SDK docs, we return formatting errors directly inside CallToolResult
		// rather than returning a protocol error to avoid hanging the MCP stream.
		return errorResult(err.Error()), nil, nil
	}
	return res, nil, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}

func successResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

func resolveSlaveID(requestSlaveID *uint8, tagSlaveID *uint8) *uint8 {
	if requestSlaveID != nil {
		return requestSlaveID
	}
	if tagSlaveID == nil {
		return nil
	}
	v := *tagSlaveID
	return &v
}
