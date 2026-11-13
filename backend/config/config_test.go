package config

import "testing"

func TestNormalizeOrigin(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{name: "render hostname", input: "finance-tracker-web.onrender.com", output: "https://finance-tracker-web.onrender.com"},
		{name: "existing scheme", input: "http://localhost:5173", output: "http://localhost:5173"},
		{name: "empty", input: "  ", output: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeOrigin(tt.input); got != tt.output {
				t.Fatalf("normalizeOrigin(%q) = %q, want %q", tt.input, got, tt.output)
			}
		})
	}
}
