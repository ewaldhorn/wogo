package ir

import (
	"strings"
	"testing"
)

// ------------------------------------------------------------------------------------------------
func TestParseGoType(t *testing.T) {
	tests := []struct {
		name    string
		goType  string
		want    Type
		wantErr string
	}{
		{"int32", "int32", TypeI32, ""},
		{"int", "int", TypeI32, ""},
		{"uint32", "uint32", TypeI32, ""},
		{"uint", "uint", TypeI32, ""},
		{"bool", "bool", TypeI32, ""},
		{"int64", "int64", TypeI64, ""},
		{"uint64", "uint64", TypeI64, ""},
		{"float32", "float32", TypeF32, ""},
		{"float64", "float64", TypeF64, ""},
		{"string rejected", "string", 0, `unsupported type "string"`},
		{"empty rejected", "", 0, `unsupported type ""`},
		{"narrow int rejected", "int16", 0, `unsupported type "int16"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGoType(tt.goType)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ParseGoType(%q) unexpected error: %v", tt.goType, err)
				}
				if got != tt.want {
					t.Errorf("ParseGoType(%q) = %v, want %v", tt.goType, got, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("ParseGoType(%q) = %v, want error %q", tt.goType, got, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ParseGoType(%q) error = %q, want it to contain %q", tt.goType, err.Error(), tt.wantErr)
			}
		})
	}
}
