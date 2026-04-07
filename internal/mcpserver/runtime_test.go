package mcpserver

import (
	"net/http"
	"testing"
)

func TestNewHTTPServerNonStreamingTimeouts(t *testing.T) {
	s := newHTTPServer(http.NewServeMux(), false)

	if s.Addr != listenAddr {
		t.Fatalf("expected addr %q, got %q", listenAddr, s.Addr)
	}
	if s.ReadHeaderTimeout != readHeaderTimeout {
		t.Fatalf("expected read header timeout %s, got %s", readHeaderTimeout, s.ReadHeaderTimeout)
	}
	if s.ReadTimeout != readTimeout {
		t.Fatalf("expected read timeout %s, got %s", readTimeout, s.ReadTimeout)
	}
	if s.WriteTimeout != writeTimeout {
		t.Fatalf("expected write timeout %s, got %s", writeTimeout, s.WriteTimeout)
	}
	if s.IdleTimeout != idleTimeout {
		t.Fatalf("expected idle timeout %s, got %s", idleTimeout, s.IdleTimeout)
	}
}

func TestNewHTTPServerStreamingDisablesWriteTimeout(t *testing.T) {
	s := newHTTPServer(http.NewServeMux(), true)
	if s.WriteTimeout != streamingWriteTimeout {
		t.Fatalf("expected streaming write timeout %s, got %s", streamingWriteTimeout, s.WriteTimeout)
	}
}
