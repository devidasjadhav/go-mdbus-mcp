package modbus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerStatusTools(s *mcp.Server, driver Driver) {
	mcp.AddTool(s,
		&mcp.Tool{
			Name:        "get-modbus-client-status",
			Description: "Get Modbus client retry and connection lifecycle status",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
			raw, err := json.MarshalIndent(driver.Status(), "", "  ")
			if err != nil {
				return errorResult(fmt.Sprintf("failed to format client status: %v", err)), nil, nil
			}
			return successResult(string(raw)), nil, nil
		},
	)
}
