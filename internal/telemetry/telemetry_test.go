package telemetry

import (
	"testing"
)

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		input  string
		want   string
		wantOK bool
	}{
		// Valid
		{"localhost:4318", "localhost:4318", true},
		{"localhost", "localhost", true},
		{"http://localhost:4318", "localhost:4318", true},
		{"http://localhost:4318/v1", "http://localhost:4318/v1", true},
		// Invalid
		{"", "", false},
		{"   ", "   ", false},
		{"http://", "http://", false},
		{"://", "://", false},
		{":4318", ":4318", false},
		{"host:notaport", "host:notaport", false},
		{"host:0", "host:0", false},
		{"host:65536", "host:65536", false},
		{"host-", "host-", false},
		// Valid: not-an-endpoint accepted
		{"not-an-endpoint", "not-an-endpoint", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := normalizeEndpoint(tt.input)
			if got != tt.want {
				t.Errorf("normalizeEndpoint(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if ok != tt.wantOK {
				t.Errorf("normalizeEndpoint(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
		})
	}
}
