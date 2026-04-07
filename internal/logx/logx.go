package logx

import (
	"log"
	"os"
)

// ConfigureStderr routes standard logger output to stderr.
// This keeps stdout clean for MCP stdio transport frames.
func ConfigureStderr() {
	log.SetOutput(os.Stderr)
}
