package main

import (
	"testing"
)

// ------------------------------------------------------------------------------------------------
func TestSplitArguments(t *testing.T) {
	tests := []struct {
		name                            string
		args                            []string
		wantInput, wantTarget, wantWasm string
		wantErr                         bool
	}{
		{"two positionals", []string{"in.go", "out.wat"}, "in.go", "out.wat", "", false},
		{"three positionals", []string{"in.go", "out.wat", "out.wasm"}, "in.go", "out.wat", "out.wasm", false},
		{"extra positionals ignored", []string{"a", "b", "c", "d"}, "a", "b", "c", false},
		{"flag after positionals rejected", []string{"in.go", "out.wat", "-v"}, "", "", "", true},
		{"long flag after positionals rejected", []string{"in.go", "out.wat", "--version"}, "", "", "", true},
		{"flag in first slot rejected", []string{"-v", "out.wat"}, "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInput, gotTarget, gotWasm, err := splitArguments(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("splitArguments(%q) err = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if gotInput != tt.wantInput || gotTarget != tt.wantTarget || gotWasm != tt.wantWasm {
				t.Errorf("splitArguments(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.args, gotInput, gotTarget, gotWasm, tt.wantInput, tt.wantTarget, tt.wantWasm)
			}
		})
	}
}
