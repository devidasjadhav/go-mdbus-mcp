package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"github.com/strowk/foxy-contexts/pkg/stdio"
	"github.com/strowk/foxy-contexts/pkg/streamable_http"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/devidasjadhav/go-mdbus-mcp/modbus"
)

// Version information set during build
var version = "dev"

func main() {
	// Ensure standard logging writes to stderr so it doesn't corrupt stdout for stdio transport
	log.SetOutput(os.Stderr)

	// Parse command-line arguments
	modbusIP := flag.String("modbus-ip", "192.168.1.22", "Modbus server IP address")
	modbusPort := flag.Int("modbus-port", 5002, "Modbus server port")
	transportFlag := flag.String("transport", "http", "Transport to use: http or stdio")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Fprintf(os.Stderr, "Modbus MCP Server v%s\n", version)
		fmt.Fprintln(os.Stderr, "https://github.com/devidasjadhav/go-mdbus-mcp")
		return
	}

	config := &modbus.Config{
		ModbusIP:   *modbusIP,
		ModbusPort: *modbusPort,
	}

	fmt.Fprintf(os.Stderr, "🚀 Modbus MCP Server v%s\n", version)
	fmt.Fprintf(os.Stderr, "📡 Connecting to Modbus server at %s:%d\n", config.ModbusIP, config.ModbusPort)
	fmt.Fprintf(os.Stderr, "🔌 Using %s transport\n", *transportFlag)

	modbusClient := modbus.NewModbusClient(config)
	fmt.Fprintln(os.Stderr, "🔧 Modbus client initialized - will connect per operation")
	fmt.Fprintln(os.Stderr, "📖 For help, visit: https://github.com/devidasjadhav/go-mdbus-mcp")

	var mcpTransport server.Transport
	if *transportFlag == "stdio" {
		mcpTransport = stdio.NewTransport()
	} else {
		mcpTransport = streamable_http.NewTransport(
			streamable_http.Endpoint{
				Hostname: "0.0.0.0",
				Port:     8080,
				Path:     "/mcp",
			})
	}

	builder := app.
		NewBuilder().
		// adding the tools to the app
		WithTool(func() fxctx.Tool { return modbus.NewReadHoldingRegistersTool(modbusClient) }).
		WithTool(func() fxctx.Tool { return modbus.NewReadCoilsTool(modbusClient) }).
		WithTool(func() fxctx.Tool { return modbus.NewWriteHoldingRegistersTool(modbusClient) }).
		WithTool(func() fxctx.Tool { return modbus.NewWriteCoilsTool(modbusClient) }).
		WithServerCapabilities(&mcp.ServerCapabilities{
			Tools: &mcp.ServerCapabilitiesTools{
				ListChanged: modbus.Ptr(false),
			},
		}).
		// setting up server
		WithName("modbus-mcp-server").
		WithVersion(version).
		WithTransport(mcpTransport).
		// Configuring fx logging to only show errors, and force zap to use stderr
		WithFxOptions(
			fx.Provide(func() *zap.Logger {
				cfg := zap.NewDevelopmentConfig()
				cfg.Level.SetLevel(zap.ErrorLevel)
				cfg.OutputPaths = []string{"stderr"}
				cfg.ErrorOutputPaths = []string{"stderr"}
				logger, _ := cfg.Build()
				return logger
			}),
			fx.Option(fx.WithLogger(
				func(logger *zap.Logger) fxevent.Logger {
					return &fxevent.ZapLogger{Logger: logger}
				},
			)),
		)

	// Add health check endpoint if we're not running in stdio mode,
	// or optionally start it in background regardless.
	if *transportFlag == "http" {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "healthy",
				"version": version,
				"service": "modbus-mcp-server",
			})
		})

		go func() {
			log.Println("🏥 Health check server starting on :8081")
			if err := http.ListenAndServe(":8081", nil); err != nil {
				log.Printf("Health check server error: %v", err)
			}
		}()
	}

	srv := builder

	err := srv.Run()
	if err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server closed")
		} else {
			log.Fatalf("Server error: %v", err)
		}
	}
}
