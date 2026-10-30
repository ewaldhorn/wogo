package compiler

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ------------------------------------------------------------------------------------------------
func TestCompiler_SumTwoInts(t *testing.T) {
	src := `package calc

func Sum(a int32, b int32) int32 {
	return a + b
}
`
	c := NewCompiler()
	wasmBytes, err := c.Compile(src)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}

	if !bytes.HasPrefix(wasmBytes, wasmMagic) {
		t.Fatalf("missing wasm magic header")
	}

	// Verify using wasmtime directly
	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "sum.wasm")
	if err := os.WriteFile(wasmPath, wasmBytes, 0644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}

	// for a = 10, b = 15,  a + b = 25
	cmd := exec.Command("wasmtime", "--invoke=Sum", wasmPath, "10", "15")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wasmtime invoke failed: %v, output: %s", err, string(out))
	}

	expected := "25"
	if !bytes.Contains(out, []byte(expected)) {
		t.Errorf("got output %q, expected containing %q", string(out), expected)
	}
}

// ------------------------------------------------------------------------------------------------
func TestCompiler_SumThreeInts(t *testing.T) {
	src := `package calc

func Sum(a int32, b int32, c int32) int32 {
	return a + b + c
}
`
	c := NewCompiler()
	wasmBytes, err := c.Compile(src)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}

	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "sum3.wasm")
	if err := os.WriteFile(wasmPath, wasmBytes, 0644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}

	// for a = 10, b = 15, c = 20, a + b + c = 45
	cmd := exec.Command("wasmtime", "--invoke=Sum", wasmPath, "10", "15", "20")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wasmtime invoke failed: %v, output: %s", err, string(out))
	}

	expected := "45"
	if !bytes.Contains(out, []byte(expected)) {
		t.Errorf("got output %q, expected containing %q", string(out), expected)
	}
}

// ------------------------------------------------------------------------------------------------
func TestCompiler_ArithmeticWithLiterals(t *testing.T) {
	src := `package calc

func Calc(a int32) int32 {
	return (a * 2) + 10 - 4
}
`
	c := NewCompiler()
	wasmBytes, err := c.Compile(src)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}

	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "calc.wasm")
	if err := os.WriteFile(wasmPath, wasmBytes, 0644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}

	// For a = 5: (5 * 2) + 10 - 4 = 16
	cmd := exec.Command("wasmtime", "--invoke=Calc", wasmPath, "5")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wasmtime invoke failed: %v, output: %s", err, string(out))
	}

	expected := "16"
	if !bytes.Contains(out, []byte(expected)) {
		t.Errorf("got output %q, expected containing %q", string(out), expected)
	}
}
