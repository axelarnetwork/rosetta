package rosetta

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_validateUrl(t *testing.T) {
	tests := []struct {
		name          string
		tendermintRPC string
		expected      string
	}{
		{
			name:          "complete url",
			tendermintRPC: "https://myhost.com:443",
			expected:      "https://myhost.com:443",
		},
		{
			name:          "no schema no port",
			tendermintRPC: "myhost.com",
			expected:      "http://myhost.com:80",
		},
		{
			name:          "http schema with no port",
			tendermintRPC: "http://myHost.com",
			expected:      "http://myhost.com:80",
		},
		{
			name:          "https schema with no port",
			tendermintRPC: "https://myHost.com",
			expected:      "https://myhost.com:443",
		},
		{
			name:          "no schema with port",
			tendermintRPC: "myHost.com:2344",
			expected:      "http://myhost.com:2344",
		},
		{
			name:          "no schema with port 443",
			tendermintRPC: "myHost.com:443",
			expected:      "https://myhost.com:443",
		},
		{
			name:          "tcp schema",
			tendermintRPC: "tcp://localhost:26657",
			expected:      "tcp://localhost:26657",
		},
		{
			name:          "localhost",
			tendermintRPC: "localhost",
			expected:      "http://localhost:80",
		},
		{
			name:          "not normalized url",
			tendermintRPC: "hTTp://tHISmyWebsite.COM",
			expected:      "http://thismywebsite.com:80",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			got, err := c.validateURL(tt.tendermintRPC)
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestParseSymbolDecimals(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []SymbolDecimal
		expectError bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:  "single entry",
			input: "uaxl:AXL:6",
			expected: []SymbolDecimal{
				{Base: "uaxl", Symbol: "AXL", Decimal: 6},
			},
		},
		{
			name:  "multiple entries",
			input: "uaxl:AXL:6,uatom:ATOM:6",
			expected: []SymbolDecimal{
				{Base: "uaxl", Symbol: "AXL", Decimal: 6},
				{Base: "uatom", Symbol: "ATOM", Decimal: 6},
			},
		},
		{
			name:  "with spaces",
			input: " uaxl : AXL : 6 , uatom : ATOM : 8 ",
			expected: []SymbolDecimal{
				{Base: "uaxl", Symbol: "AXL", Decimal: 6},
				{Base: "uatom", Symbol: "ATOM", Decimal: 8},
			},
		},
		{
			name:  "trailing comma",
			input: "uaxl:AXL:6,",
			expected: []SymbolDecimal{
				{Base: "uaxl", Symbol: "AXL", Decimal: 6},
			},
		},
		{
			name:        "invalid format - missing parts",
			input:       "uaxl:AXL",
			expectError: true,
		},
		{
			name:        "invalid format - too many parts",
			input:       "uaxl:AXL:6:extra",
			expectError: true,
		},
		{
			name:        "invalid decimal",
			input:       "uaxl:AXL:notanumber",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSymbolDecimals(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
