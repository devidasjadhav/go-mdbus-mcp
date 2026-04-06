package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/devidasjadhav/go-mdbus-mcp/modbus"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "dev"

func main() {
	// Standard logging writes to stderr so it doesn't corrupt stdout for stdio transport
	log.SetOutput(os.Stderr)

	modbusIP := flag.String("modbus-ip", "192.168.1.22", "Modbus server IP address")
	modbusPort := flag.Int("modbus-port", 5002, "Modbus server port")
	transportFlag := flag.String("transport", "streamable", "Transport to use: stdio, sse, or streamable")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Fprintf(os.Stderr, "Modbus MCP Server v%s\n", version)
		fmt.Fprintln(os.Stderr, "https://github.com/devidasjadhav/go-mdbus-mcp")
		return
	}

	fmt.Fprintf(os.Stderr, "🚀 Modbus MCP Server v%s\n", version)
	fmt.Fprintf(os.Stderr, "📡 Connecting to Modbus server at %s:%d\n", *modbusIP, *modbusPort)
	fmt.Fprintf(os.Stderr, "🔌 Using %s transport\n", *transportFlag)

	// Create Modbus client
	modbusClient := modbus.NewModbusClient(&modbus.Config{
		ModbusIP:   *modbusIP,
		ModbusPort: *modbusPort,
	})

	// Create standard MCP server instance
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "modbus-mcp-server",
		Version: version,
	}, nil)

	// Register tools natively with the SDK
	modbus.RegisterTools(s, modbusClient)

	// Start Transport
	ctx := context.Background()

	switch *transportFlag {
	case "stdio":
		fmt.Fprintln(os.Stderr, "Starting stdio transport...")
		if err := s.Run(ctx, &mcp.StdioTransport{}); err != nil {
			log.Fatalf("Server error: %v", err)
		}

	case "sse":
		fmt.Fprintln(os.Stderr, "Starting SSE transport on :8080...")
		sseHandler := mcp.NewSSEHandler(func(req *http.Request) *mcp.Server { return s }, nil)

		mux := http.NewServeMux()
		mux.Handle("/sse", sseHandler)
		mux.Handle("/message", sseHandler)

		setupHealthCheck(mux)

		if err := http.ListenAndServe("0.0.0.0:8080", mux); err != nil {
			log.Fatalf("Server error: %v", err)
		}

	default: // "streamable" or anything else
		fmt.Fprintln(os.Stderr, "Starting Streamable HTTP transport on :8080...")
		streamableHandler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server { return s }, nil)

		mux := http.NewServeMux()
		mux.Handle("/mcp", streamableHandler)

		setupHealthCheck(mux)

		if err := http.ListenAndServe("0.0.0.0:8080", mux); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

func setupHealthCheck(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"version": version,
			"service": "modbus-mcp-server",
		})
	})
}
