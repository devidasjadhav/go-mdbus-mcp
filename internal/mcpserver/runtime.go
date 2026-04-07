package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func Run(ctx context.Context, transport string, server *mcp.Server, version string) error {
	switch transport {
	case "stdio":
		return server.Run(ctx, &mcp.StdioTransport{})

	case "sse":
		sseHandler := mcp.NewSSEHandler(func(req *http.Request) *mcp.Server { return server }, nil)

		mux := http.NewServeMux()
		mux.Handle("/sse", sseHandler)
		mux.Handle("/message", sseHandler)
		setupHealthCheck(mux, version)

		httpServer := &http.Server{Addr: "0.0.0.0:8080", Handler: mux}
		go shutdownOnContextCancel(ctx, httpServer)

		err := httpServer.ListenAndServe()
		if err == http.ErrServerClosed {
			return nil
		}
		return err

	default: // streamable
		streamableHandler := mcp.NewStreamableHTTPHandler(
			func(req *http.Request) *mcp.Server { return server },
			&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
		)

		mux := http.NewServeMux()
		mux.Handle("/mcp", streamableHandler)
		setupHealthCheck(mux, version)

		httpServer := &http.Server{Addr: "0.0.0.0:8080", Handler: mux}
		go shutdownOnContextCancel(ctx, httpServer)

		err := httpServer.ListenAndServe()
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func shutdownOnContextCancel(ctx context.Context, httpServer *http.Server) {
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func setupHealthCheck(mux *http.ServeMux, version string) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"version": version,
			"service": "modbus-mcp-server",
		})
	})
}
