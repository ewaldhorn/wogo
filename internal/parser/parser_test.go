package parser

import (
	"strings"
	"testing"
)

// ------------------------------------------------------------------------------------------------
func TestParse_InvalidSourceReturnsError(t *testing.T) {
	p := &Parser{}

	// Must return an error, not panic anymore
	wat, err := p.Parse("func broken( {")
	if err == nil {
		t.Fatalf("expected error for invalid Go source, got nil (wat=%q)", wat)
	}
	if wat != "" {
		t.Errorf("expected empty WAT on error, got %q", wat)
	}
}

// ------------------------------------------------------------------------------------------------
func TestParse_ValidSource(t *testing.T) {
	p := &Parser{}

	wat, err := p.Parse("package calc\n\nfunc Add(a int32, b int32) int32 { return a + b }\n")
	if err != nil {
		t.Fatalf("expected no error for valid source, got %v", err)
	}
	if !strings.Contains(wat, `(func $Add`) {
		t.Errorf("expected WAT to declare func $Add, got:\n%s", wat)
	}
}
