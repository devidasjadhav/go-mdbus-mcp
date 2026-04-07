package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	appconfig "github.com/devidasjadhav/go-mdbus-mcp/internal/config"
	"github.com/devidasjadhav/go-mdbus-mcp/internal/logx"
	"github.com/devidasjadhav/go-mdbus-mcp/internal/mcpserver"
	"github.com/devidasjadhav/go-mdbus-mcp/modbus"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "dev"

func main() {
	// Standard logging writes to stderr so it doesn't corrupt stdout for stdio transport
	logx.ConfigureStderr()

	configPath := flag.String("config", "", "Path to YAML/JSON config file")
	tagMapCSV := flag.String("tag-map-csv", "", "Path to CSV tag mapping file")
	modbusDriver := flag.String("modbus-driver", "goburrow", "Modbus driver to use: goburrow|simonvetter")
	modbusMode := flag.String("modbus-mode", "tcp", "Modbus mode to use: tcp|rtu")
	serialPort := flag.String("serial-port", "", "Serial device path for RTU mode (e.g. /dev/ttyUSB0)")
	baudRate := flag.Int("baud-rate", 9600, "RTU baud rate")
	dataBits := flag.Int("data-bits", 8, "RTU data bits")
	parity := flag.String("parity", "N", "RTU parity: N|E|O")
	stopBits := flag.Int("stop-bits", 1, "RTU stop bits")
	modbusIP := flag.String("modbus-ip", "192.168.1.22", "Modbus server IP address")
	modbusPort := flag.Int("modbus-port", 5002, "Modbus server port")
	modbusTimeout := flag.Duration("modbus-timeout", 10*time.Second, "Modbus request timeout (e.g. 10s)")
	modbusIdleTimeout := flag.Duration("modbus-idle-timeout", 2*time.Second, "Modbus TCP idle timeout before proactive close (e.g. 2s)")
	modbusRetryAttempts := flag.Int("modbus-retry-attempts", 3, "Modbus retry attempts for transient errors")
	modbusRetryBackoff := flag.Duration("modbus-retry-backoff", 150*time.Millisecond, "Initial retry backoff for transient errors")
	modbusRetryOnWrite := flag.Bool("modbus-retry-on-write", false, "Allow retries on write operations")
	modbusReconnectPerOp := flag.Bool("modbus-reconnect-per-operation", true, "Reconnect Modbus TCP client before each operation")
	modbusConnectionPoolSize := flag.Int("modbus-connection-pool-size", 1, "Number of Modbus TCP client connections to maintain (read ops can be load-balanced)")
	modbusCircuitTripAfter := flag.Int("modbus-circuit-trip-after", 3, "Consecutive failures before opening circuit")
	modbusCircuitOpenFor := flag.Duration("modbus-circuit-open-for", 2*time.Second, "Duration to keep circuit open after trip")
	mockMode := flag.Bool("mock-mode", false, "Run without real Modbus device using in-memory mock client")
	mockRegisters := flag.Int("mock-registers", 1024, "Mock holding register count (mock-mode)")
	mockCoils := flag.Int("mock-coils", 1024, "Mock coil count (mock-mode)")
	transportFlag := flag.String("transport", "streamable", "Transport to use: stdio, sse, or streamable")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	runtimeOpts := appconfig.RuntimeOptions{
		ModbusDriver:             *modbusDriver,
		ModbusMode:               *modbusMode,
		SerialPort:               *serialPort,
		BaudRate:                 *baudRate,
		DataBits:                 *dataBits,
		Parity:                   *parity,
		StopBits:                 *stopBits,
		ModbusIP:                 *modbusIP,
		ModbusPort:               *modbusPort,
		ModbusTimeout:            *modbusTimeout,
		ModbusIdleTimeout:        *modbusIdleTimeout,
		ModbusRetryAttempts:      *modbusRetryAttempts,
		ModbusRetryBackoff:       *modbusRetryBackoff,
		ModbusRetryOnWrite:       *modbusRetryOnWrite,
		ModbusReconnectPerOp:     *modbusReconnectPerOp,
		ModbusConnectionPoolSize: *modbusConnectionPoolSize,
		CircuitTripAfter:         *modbusCircuitTripAfter,
		CircuitOpenFor:           *modbusCircuitOpenFor,
		MockMode:                 *mockMode,
		MockRegisters:            *mockRegisters,
		MockCoils:                *mockCoils,
		Transport:                *transportFlag,
	}

	setFlags := map[string]bool{}
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	var fileCfg *appconfig.AppConfig
	if strings.TrimSpace(*configPath) != "" {
		cfg, err := appconfig.LoadAppConfig(*configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		fileCfg = cfg
		if err := appconfig.ApplyConfigOverrides(fileCfg, setFlags, &runtimeOpts); err != nil {
			log.Fatalf("Invalid config value: %v", err)
		}
		if fileCfg.TagMapCSV != nil && !setFlags["tag-map-csv"] {
			*tagMapCSV = resolveConfigRelativePath(*configPath, *fileCfg.TagMapCSV)
		}
	}

	if err := appconfig.ValidateRuntimeOptions(&runtimeOpts); err != nil {
		log.Fatalf("Invalid runtime configuration: %v", err)
	}

	if *showVersion {
		fmt.Fprintf(os.Stderr, "Modbus MCP Server v%s\n", version)
		fmt.Fprintln(os.Stderr, "https://github.com/devidasjadhav/go-mdbus-mcp")
		return
	}

	fmt.Fprintf(os.Stderr, "🚀 Modbus MCP Server v%s\n", version)
	fmt.Fprintf(os.Stderr, "🧩 Driver=%s Mode=%s\n", runtimeOpts.ModbusDriver, runtimeOpts.ModbusMode)
	fmt.Fprintf(os.Stderr, "📡 Connecting to Modbus server at %s:%d\n", runtimeOpts.ModbusIP, runtimeOpts.ModbusPort)
	fmt.Fprintf(os.Stderr, "🔌 Using %s transport\n", runtimeOpts.Transport)
	if runtimeOpts.ModbusMode == "rtu" {
		fmt.Fprintf(os.Stderr, "🔌 RTU serial settings: port=%s baud=%d dataBits=%d parity=%s stopBits=%d\n", runtimeOpts.SerialPort, runtimeOpts.BaudRate, runtimeOpts.DataBits, runtimeOpts.Parity, runtimeOpts.StopBits)
	}
	if runtimeOpts.MockMode {
		fmt.Fprintf(os.Stderr, "🧪 Mock mode enabled (registers=%d, coils=%d)\n", runtimeOpts.MockRegisters, runtimeOpts.MockCoils)
	}

	// Create Modbus driver
	driver, err := modbus.NewDriver(&modbus.Config{
		Driver:                   runtimeOpts.ModbusDriver,
		Mode:                     runtimeOpts.ModbusMode,
		SerialPort:               runtimeOpts.SerialPort,
		BaudRate:                 runtimeOpts.BaudRate,
		DataBits:                 runtimeOpts.DataBits,
		Parity:                   runtimeOpts.Parity,
		StopBits:                 runtimeOpts.StopBits,
		ModbusIP:                 runtimeOpts.ModbusIP,
		ModbusPort:               runtimeOpts.ModbusPort,
		Timeout:                  runtimeOpts.ModbusTimeout,
		IdleTimeout:              runtimeOpts.ModbusIdleTimeout,
		DefaultSlaveID:           1,
		RetryAttempts:            runtimeOpts.ModbusRetryAttempts,
		RetryBackoff:             runtimeOpts.ModbusRetryBackoff,
		RetryOnWrite:             runtimeOpts.ModbusRetryOnWrite,
		ReconnectPerOp:           runtimeOpts.ModbusReconnectPerOp,
		ReconnectPerOpConfigured: true,
		ConnectionPoolSize:       runtimeOpts.ModbusConnectionPoolSize,
		CircuitTripAfter:         runtimeOpts.CircuitTripAfter,
		CircuitOpenFor:           runtimeOpts.CircuitOpenFor,
		UseMock:                  runtimeOpts.MockMode,
		MockRegisters:            runtimeOpts.MockRegisters,
		MockCoils:                runtimeOpts.MockCoils,
	})
	if err != nil {
		log.Fatalf("failed to create modbus driver: %v", err)
	}
	defer func() {
		if err := driver.Close(); err != nil {
			log.Printf("failed to close modbus driver: %v", err)
		}
	}()

	// Create standard MCP server instance
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "modbus-mcp-server",
		Version: version,
	}, nil)

	writePolicy, err := modbus.LoadWritePolicy(appconfig.ToWritePolicyOverrides(fileCfg))
	if err != nil {
		log.Fatalf("Invalid write policy configuration: %v", err)
	}

	tagMap, err := appconfig.ToTagMap(fileCfg, *tagMapCSV)
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
	emitSecurityWarnings(runtimeOpts.ModbusPort, runtimeOpts.MockMode, writePolicy)

	// Register tools natively with the SDK
	modbus.RegisterTools(s, driver, writePolicy, tagMap)

	// Start transport with graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runErr := mcpserver.Run(ctx, runtimeOpts.Transport, s, version)

	if runErr != nil {
		log.Printf("Server error: %v", runErr)
	}
}

func resolveConfigRelativePath(configPath string, candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return ""
	}
	if filepath.IsAbs(candidate) {
		return candidate
	}
	return filepath.Join(filepath.Dir(configPath), candidate)
}

func emitSecurityWarnings(modbusPort int, mockMode bool, writePolicy *modbus.WritePolicy) {
	if modbusPort > 0 && modbusPort < 1024 {
		fmt.Fprintf(os.Stderr, "⚠️  Privileged port %d may require elevated permissions; prefer non-root ports like 1502 when possible.\n", modbusPort)
	}

	if !mockMode {
		if currentUser, err := user.Current(); err == nil && currentUser.Uid == "0" {
			fmt.Fprintln(os.Stderr, "⚠️  Server is running as root. For production, run as a dedicated non-root service account.")
		}
	}

	if writePolicy != nil && writePolicy.Enabled() && !writePolicy.HasAllowlist() {
		fmt.Fprintln(os.Stderr, "⚠️  Writes are enabled without an allowlist. Consider MODBUS_WRITE_ALLOWLIST for safer operation.")
	}
}
