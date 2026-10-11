package telemetry

import (
	"testing"
)

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		// Bare host:port
		{"bare host:port", "localhost:4318", "localhost:4318", true},
		{"bare host:port with ip", "127.0.0.1:4318", "127.0.0.1:4318", true},

		// http:// prefixed
		{"http:// prefix", "http://localhost:4318", "localhost:4318", true},
		{"http:// with ip", "http://127.0.0.1:4318", "127.0.0.1:4318", true},

		// https:// prefixed
		{"https:// prefix", "https://localhost:4318", "localhost:4318", true},
		{"https:// with ip", "https://127.0.0.1:4318", "127.0.0.1:4318", true},

		// Full URL with path (should be returned as-is)
		{"full http URL with path", "http://localhost:4318/v1/traces", "http://localhost:4318/v1/traces", true},
		{"full https URL with path", "https://otel-collector.example.com:4317/v1/traces", "https://otel-collector.example.com:4317/v1/traces", true},
		// Full URL without path (scheme + host:port only - strip scheme)
		{"https URL no path", "https://otel-collector.example.com:4317", "otel-collector.example.com:4317", true},

		// Empty
		{"empty", "", "", false},

		// Garbage
		{"garbage - not an endpoint", "not an endpoint", "not an endpoint", false},
		{"garbage - just http://", "http://", "http://", false},
		{"garbage - just ://", "://", "://", false},
		{"garbage - just colon", ":", ":", false},
		{"garbage - no host", ":4318", ":4318", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOK := normalizeEndpoint(tt.input)
			if got != tt.want {
				t.Errorf("normalizeEndpoint(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if gotOK != tt.wantOK {
				t.Errorf("normalizeEndpoint(%q) ok = %v, want %v", tt.input, gotOK, tt.wantOK)
			}
		})
	}
}
