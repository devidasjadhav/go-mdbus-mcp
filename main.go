package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/streamable_http"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/devidasjadhav/go-mdbus-mcp/modbus"
)

// Version information set during build
var version = "dev"

func main() {
	// Parse command-line arguments
	modbusIP := flag.String("modbus-ip", "192.168.1.22", "Modbus server IP address")
	modbusPort := flag.Int("modbus-port", 5002, "Modbus server port")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("Modbus MCP Server v%s\n", version)
		fmt.Println("https://github.com/devidasjadhav/go-mdbus-mcp")
		return
	}

	config := &modbus.Config{
		ModbusIP:   *modbusIP,
		ModbusPort: *modbusPort,
	}

	fmt.Printf("🚀 Modbus MCP Server v%s\n", version)
	fmt.Printf("📡 Connecting to Modbus server at %s:%d\n", config.ModbusIP, config.ModbusPort)

	modbusClient := modbus.NewModbusClient(config)
	fmt.Println("🔧 Modbus client initialized - will connect per operation")
	fmt.Println("📖 For help, visit: https://github.com/devidasjadhav/go-mdbus-mcp")

	server := app.
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
		WithVersion("0.0.1").
		WithTransport(
			streamable_http.NewTransport(
				streamable_http.Endpoint{
					Hostname: "localhost",
					Port:     8080,
					Path:     "/mcp",
				}),
		).
		// Configuring fx logging to only show errors
		WithFxOptions(
			fx.Provide(func() *zap.Logger {
				cfg := zap.NewDevelopmentConfig()
				cfg.Level.SetLevel(zap.ErrorLevel)
				logger, _ := cfg.Build()
				return logger
			}),
			fx.Option(fx.WithLogger(
				func(logger *zap.Logger) fxevent.Logger {
					return &fxevent.ZapLogger{Logger: logger}
				},
			)),
		)

	// Add health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"version": version,
			"service": "modbus-mcp-server",
		})
	})

	// Start health check server in a goroutine
	go func() {
		log.Println("🏥 Health check server starting on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Printf("Health check server error: %v", err)
		}
	}()

	err := server.Run()
	if err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server closed")
		} else {
			log.Fatalf("Server error: %v", err)
		}
	}
}
