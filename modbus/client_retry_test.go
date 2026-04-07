package modbus

import (
	"errors"
	"testing"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestShouldRetryError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "eof", err: errors.New("EOF"), want: true},
		{name: "eof_lower", err: errors.New("eof"), want: true},
		{name: "broken_pipe", err: errors.New("write: broken pipe"), want: true},
		{name: "timeout_net_error", err: timeoutErr{}, want: true},
		{name: "wrapped_timeout", err: errors.New("wrap: " + timeoutErr{}.Error()), want: true},
		{name: "random", err: errors.New("permission denied"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRetryError(tc.err)
			if got != tc.want {
				t.Fatalf("shouldRetryError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
