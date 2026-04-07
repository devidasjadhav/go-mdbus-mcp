package modbus

import (
	"context"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type driverSelector interface {
	SelectDriverForOp(allowRetry bool) Driver
}

// executeTool is a helper to run thread-safe operations on the Modbus client
// and handle any protocol errors.
func executeTool(ctx context.Context, driver Driver, slaveID *uint8, allowRetry bool, operation func(active Driver) (*mcp.CallToolResult, error)) (*mcp.CallToolResult, any, error) {
	targetSlaveID := uint8(1)
	if slaveID != nil {
		targetSlaveID = *slaveID
	}

	runDriver := driver
	if selector, ok := driver.(driverSelector); ok {
		if selected := selector.SelectDriverForOp(allowRetry); selected != nil {
			runDriver = selected
		}
	}

	start := time.Now()
	res, err := runDriver.Execute(ctx, targetSlaveID, allowRetry, func() (*mcp.CallToolResult, error) {
		return operation(runDriver)
	})
	elapsed := time.Since(start)
	if err != nil {
		log.Printf("modbus op failed driver=%s mode=%s slave_id=%d allow_retry=%t duration=%s err=%v", runDriver.DriverName(), runDriver.TransportMode(), targetSlaveID, allowRetry, elapsed, err)
	} else {
		log.Printf("modbus op success driver=%s mode=%s slave_id=%d allow_retry=%t duration=%s", runDriver.DriverName(), runDriver.TransportMode(), targetSlaveID, allowRetry, elapsed)
	}
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
