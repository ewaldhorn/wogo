package ir

import (
	"strings"
	"testing"
)

// ------------------------------------------------------------------------------------------------
// Sanity checker to make sure we can do basic parsing, at least.
// If this smoke test fails, there are big problems.
func TestBuildModule_Simple(t *testing.T) {
	src := `package calc

const PI = 3.141592653589793

func DegreesToRads(deg float64) float64 {
	const LocalConst = 180.0
	return deg * (PI / LocalConst)
}
`
	mod, err := BuildModule(src)
	if err != nil {
		t.Fatalf("BuildModule failed: %v", err)
	}

	if len(mod.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(mod.Functions))
	}

	fn := mod.Functions[0]
	if fn.Name != "DegreesToRads" {
		t.Errorf("expected function name DegreesToRads, got %s", fn.Name)
	}

	if len(fn.Params) != 1 || fn.Params[0].Name != "deg" || fn.Params[0].Type != TypeF64 {
		t.Errorf("unexpected params: %+v", fn.Params)
	}

	if len(fn.Results) != 1 || fn.Results[0] != TypeF64 {
		t.Errorf("unexpected results: %+v", fn.Results)
	}

	if len(fn.Instructions) == 0 {
		t.Fatalf("expected instructions, got none")
	}

	last := fn.Instructions[len(fn.Instructions)-1]
	if last.Op != OpReturn {
		t.Errorf("expected last instruction to be OpReturn, got %v", last.Op)
	}
}

// ------------------------------------------------------------------------------------------------
func TestBuildModule_Invalid(t *testing.T) {
	_, err := BuildModule("broken syntax {")
	if err == nil {
		t.Error("expected syntax error, got nil")
	}
}

// ------------------------------------------------------------------------------------------------
func TestBuildModule_GoInt(t *testing.T) {
	mod, err := BuildModule("package calc\n\nfunc Add(a int, b int) int { return a + b }\n")
	if err != nil {
		t.Fatalf("expected int to compile, got %v", err)
	}

	fn := mod.Functions[0]
	for _, p := range fn.Params {
		if p.Type != TypeI32 {
			t.Errorf("param %s: expected TypeI32, got %v", p.Name, p.Type)
		}
	}
	if len(fn.Results) != 1 || fn.Results[0] != TypeI32 {
		t.Errorf("unexpected results: %+v", fn.Results)
	}
}

// ------------------------------------------------------------------------------------------------
func TestBuildModule_UnsupportedType(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			"string param",
			"package calc\n\nfunc Bad(a string) string { return a }\n",
			`unsupported type "string"`,
		},
		{
			"string result",
			"package calc\n\nfunc Bad() string { return \"hi\" }\n",
			`unsupported type "string"`,
		},
		{
			"string const",
			"package calc\n\nconst x string = \"hi\"\n\nfunc F() int32 { return x }\n",
			`unsupported type "string"`,
		},
		{
			"pointer param",
			"package calc\n\nfunc F(a *int32) int32 { return a }\n",
			"unsupported type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildModule(tt.src)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
