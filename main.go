package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/devidasjadhav/go-mdbus-mcp/modbus"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "dev"

func main() {
	// Standard logging writes to stderr so it doesn't corrupt stdout for stdio transport
	log.SetOutput(os.Stderr)

	configPath := flag.String("config", "", "Path to YAML/JSON config file")
	modbusIP := flag.String("modbus-ip", "192.168.1.22", "Modbus server IP address")
	modbusPort := flag.Int("modbus-port", 5002, "Modbus server port")
	modbusTimeout := flag.Duration("modbus-timeout", 10*time.Second, "Modbus request timeout (e.g. 10s)")
	modbusIdleTimeout := flag.Duration("modbus-idle-timeout", 2*time.Second, "Modbus TCP idle timeout before proactive close (e.g. 2s)")
	modbusRetryAttempts := flag.Int("modbus-retry-attempts", 3, "Modbus retry attempts for transient errors")
	modbusRetryBackoff := flag.Duration("modbus-retry-backoff", 150*time.Millisecond, "Initial retry backoff for transient errors")
	modbusRetryOnWrite := flag.Bool("modbus-retry-on-write", false, "Allow retries on write operations")
	modbusCircuitTripAfter := flag.Int("modbus-circuit-trip-after", 3, "Consecutive failures before opening circuit")
	modbusCircuitOpenFor := flag.Duration("modbus-circuit-open-for", 2*time.Second, "Duration to keep circuit open after trip")
	transportFlag := flag.String("transport", "streamable", "Transport to use: stdio, sse, or streamable")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	setFlags := map[string]bool{}
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	var fileCfg *AppConfig
	if strings.TrimSpace(*configPath) != "" {
		cfg, err := loadAppConfig(*configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		fileCfg = cfg
		if err := applyConfigOverrides(fileCfg, setFlags, modbusIP, modbusPort, modbusTimeout, modbusIdleTimeout, modbusRetryAttempts, modbusRetryBackoff, modbusRetryOnWrite, modbusCircuitTripAfter, modbusCircuitOpenFor, transportFlag); err != nil {
			log.Fatalf("Invalid config value: %v", err)
		}
	}

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
		ModbusIP:         *modbusIP,
		ModbusPort:       *modbusPort,
		Timeout:          *modbusTimeout,
		IdleTimeout:      *modbusIdleTimeout,
		DefaultSlaveID:   1,
		RetryAttempts:    *modbusRetryAttempts,
		RetryBackoff:     *modbusRetryBackoff,
		RetryOnWrite:     *modbusRetryOnWrite,
		CircuitTripAfter: *modbusCircuitTripAfter,
		CircuitOpenFor:   *modbusCircuitOpenFor,
	})
	defer func() {
		if err := modbusClient.Close(); err != nil {
			log.Printf("failed to close modbus client: %v", err)
		}
	}()

	// Create standard MCP server instance
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "modbus-mcp-server",
		Version: version,
	}, nil)

	writePolicy, err := modbus.LoadWritePolicy(toWritePolicyOverrides(fileCfg))
	if err != nil {
		log.Fatalf("Invalid write policy configuration: %v", err)
	}

	tagMap, err := toTagMap(fileCfg)
	if err != nil {
		log.Fatalf("Invalid tag configuration: %v", err)
	}
	if tagMap != nil {
		fmt.Fprintf(os.Stderr, "🏷️  Loaded %d configured tags\n", len(tagMap.List()))
	}
	if writePolicy.Enabled() {
		fmt.Fprintln(os.Stderr, "✍️  Modbus writes are ENABLED by policy")
	} else {
		fmt.Fprintln(os.Stderr, "🔒 Modbus writes are DISABLED by default (set MODBUS_WRITES_ENABLED=true to allow writes)")
	}

	// Register tools natively with the SDK
	modbus.RegisterTools(s, modbusClient, writePolicy, tagMap)

	// Start transport with graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var runErr error

	switch *transportFlag {
	case "stdio":
		fmt.Fprintln(os.Stderr, "Starting stdio transport...")
		runErr = s.Run(ctx, &mcp.StdioTransport{})

	case "sse":
		fmt.Fprintln(os.Stderr, "Starting SSE transport on :8080...")
		sseHandler := mcp.NewSSEHandler(func(req *http.Request) *mcp.Server { return s }, nil)

		mux := http.NewServeMux()
		mux.Handle("/sse", sseHandler)
		mux.Handle("/message", sseHandler)

		setupHealthCheck(mux)
		httpServer := &http.Server{
			Addr:    "0.0.0.0:8080",
			Handler: mux,
		}

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				log.Printf("SSE HTTP shutdown error: %v", err)
			}
		}()

		runErr = httpServer.ListenAndServe()
		if runErr == http.ErrServerClosed {
			runErr = nil
		}

	default: // "streamable" or anything else
		fmt.Fprintln(os.Stderr, "Starting Streamable HTTP transport on :8080...")
		streamableHandler := mcp.NewStreamableHTTPHandler(
			func(req *http.Request) *mcp.Server { return s },
			&mcp.StreamableHTTPOptions{
				Stateless:    true,
				JSONResponse: true,
			},
		)

		mux := http.NewServeMux()
		mux.Handle("/mcp", streamableHandler)

		setupHealthCheck(mux)
		httpServer := &http.Server{
			Addr:    "0.0.0.0:8080",
			Handler: mux,
		}

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				log.Printf("Streamable HTTP shutdown error: %v", err)
			}
		}()

		runErr = httpServer.ListenAndServe()
		if runErr == http.ErrServerClosed {
			runErr = nil
		}
	}

	if runErr != nil {
		log.Printf("Server error: %v", runErr)
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
