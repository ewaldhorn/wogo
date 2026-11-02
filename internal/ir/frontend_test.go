package ir

import (
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
