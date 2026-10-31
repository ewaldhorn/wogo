package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"wogo/internal/compiler"
	"wogo/internal/parser"
)

// ------------------------------------------------------------------------------------------------
// run parses CLI arguments, locates test specification files, and coordinates
// test execution across all files.
//
// TODO: Add some colour for failures.
func run() error {
	var wasmMode bool
	var watMode bool
	flag.BoolVar(&wasmMode, "wasm", false, "run tests against compiled wasm binary")
	flag.BoolVar(&watMode, "wat", false, "run tests against generated wat via wat2wasm")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [-wasm|-wat] <test-file-or-dir>...\n", os.Args[0])
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		return errors.New("at least one test specification file or directory is required")
	}

	files, err := findAllTestFiles(flag.Args())
	if err != nil {
		return err
	}

	runner := &Runner{
		ForceWasm: wasmMode,
		ForceWat:  watMode,
	}

	return runner.RunFiles(context.Background(), files)
}

// ------------------------------------------------------------------------------------------------
// Runner totals execution results across all tests.
type Runner struct {
	TotalPassed int
	TotalFailed int
	FilesFailed int
	ForceWasm   bool
	ForceWat    bool
}

// ------------------------------------------------------------------------------------------------
// RunFiles executes all given test files and displays a summary at the end.
func (r *Runner) RunFiles(ctx context.Context, files []string) error {
	fmt.Println()
	for _, file := range files {
		fmt.Printf("=== RUN   %s\n", file)
		passed, failed, err := r.RunFile(ctx, file)
		r.TotalPassed += passed
		r.TotalFailed += failed
		if err != nil {
			r.FilesFailed++
			fmt.Printf("--- FAIL: %s (%v)\n\n", file, err)
		} else {
			fmt.Printf("--- PASS: %s\n\n", file)
		}
	}

	if len(files) > 1 {
		r.printSummary(len(files))
	}

	if r.FilesFailed > 0 || r.TotalFailed > 0 {
		return fmt.Errorf("%d test(s) failed across %d file(s)", r.TotalFailed, r.FilesFailed)
	}

	return nil
}

// ------------------------------------------------------------------------------------------------
// RunFile parses and executes a single test.
func (r *Runner) RunFile(ctx context.Context, path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("open specification: %w", err)
	}
	defer f.Close()

	spec, err := ParseSpec(f)
	if err != nil {
		return 0, 0, fmt.Errorf("parse specification: %w", err)
	}

	return r.RunSpec(ctx, spec)
}

// ------------------------------------------------------------------------------------------------
func (r *Runner) printSummary(totalFiles int) {
	passedFiles := totalFiles - r.FilesFailed
	fmt.Println("========================================")
	fmt.Printf("Summary: %d/%d files passed (%d assertions passed, %d failed)\n\n",
		passedFiles, totalFiles, r.TotalPassed, r.TotalFailed)
}

// ------------------------------------------------------------------------------------------------
// RunSpec compiles and executes every test case through the wasmtime CLI.
func (r *Runner) RunSpec(ctx context.Context, spec *Spec) (int, int, error) {
	isWasm := r.ForceWasm || (!r.ForceWat && strings.HasSuffix(strings.ToLower(spec.Output), ".wasm"))

	var wasmPath string
	var cleanup func()

	if isWasm {
		// Compile Go source directly to WASM using the internal compiler
		if strings.HasSuffix(strings.ToLower(spec.Output), ".wat") {
			wasmPath = strings.TrimSuffix(spec.Output, filepath.Ext(spec.Output)) + ".wasm"
		} else {
			wasmPath = spec.Output
		}
		if err := compileGoToWasm(spec.Input, wasmPath); err != nil {
			return 0, 0, err
		}
	} else {
		// Compile Go source to WAT, then convert WAT to WASM via wat2wasm (reference compiler output)
		if err := ensureOutput(spec); err != nil {
			return 0, 0, err
		}

		var err error
		wasmPath, cleanup, err = compileWatToWasm(ctx, spec.Output)
		if err != nil {
			return 0, 0, err
		}
		defer cleanup()
	}

	passed, failed := 0, 0
	for i, tc := range spec.Tests {
		if err := executeTestCase(ctx, wasmPath, tc); err != nil {
			failed++
			fmt.Printf("  FAIL [%d] %s(%s): %v\n",
				i+1, tc.Command, strings.Join(tc.Arguments, ", "), err)
		} else {
			passed++
			fmt.Printf("  PASS [%d] %s(%s) -> %s\n",
				i+1, tc.Command, strings.Join(tc.Arguments, ", "),
				strings.Join(tc.Expected, ", "))
		}
	}

	fmt.Printf("  %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		return passed, failed, fmt.Errorf("%d test(s) failed", failed)
	}

	return passed, failed, nil
}

// ------------------------------------------------------------------------------------------------
// RunSpec is a convenience wrapper for running a spec with default settings.
func RunSpec(ctx context.Context, spec *Spec) (int, int, error) {
	runner := &Runner{}
	return runner.RunSpec(ctx, spec)
}

// ------------------------------------------------------------------------------------------------
// executeTestCase runs a test case through wasmtime and verifies the results.
func executeTestCase(ctx context.Context, wasmPath string, tc TestCase) error {
	actual, err := invokeWasmtime(ctx, wasmPath, tc.Command, tc.Arguments)
	if err != nil {
		return err
	}

	return verifyResults(actual, tc.Expected)
}

// ------------------------------------------------------------------------------------------------
// invokeWasmtime invokes an exported function in a WASM module using wasmtime.
func invokeWasmtime(ctx context.Context, wasmPath, command string, args []string) ([]string, error) {
	// The equals-sign form (--invoke=Name) is required for wasmtime to print the return value.
	cmdArgs := append([]string{"--invoke=" + command, wasmPath}, args...)
	cmd := exec.CommandContext(ctx, "wasmtime", cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("wasmtime invoke failed: %w\nstderr: %s", err, stderr.String())
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return nil, nil
	}

	return strings.Fields(out), nil
}

// ------------------------------------------------------------------------------------------------
// verifyResults verifies that actual command output match the expected values.
func verifyResults(actual, expected []string) error {
	if len(actual) != len(expected) {
		if len(actual) == 0 {
			return fmt.Errorf("function returned no values, want %d", len(expected))
		}
		return fmt.Errorf("got %d result(s) %v, want %d", len(actual), actual, len(expected))
	}

	for i := range expected {
		if !matchesResult(actual[i], expected[i]) {
			return fmt.Errorf("result %d: got %s, want %s", i, actual[i], expected[i])
		}
	}

	return nil
}

func matchesResult(actual, expected string) bool {
	if actual == expected {
		return true
	}

	// WebAssembly represents booleans as i32 (0 for false, 1 for true)
	if (actual == "0" && expected == "false") || (actual == "false" && expected == "0") {
		return true
	}
	if (actual == "1" && expected == "true") || (actual == "true" && expected == "1") {
		return true
	}

	return false
}

// ------------------------------------------------------------------------------------------------
// compileWatToWasm uses WABT's wat2wasm to convert a .wat file into a temporary .wasm file.
func compileWatToWasm(ctx context.Context, watPath string) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "spectest-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	wasmPath := filepath.Join(tmpDir, "module.wasm")
	cmd := exec.CommandContext(ctx, "wat2wasm", watPath, "-o", wasmPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		if errors.Is(err, exec.ErrNotFound) {
			return "", nil, errors.New("wat2wasm not found in PATH; install WABT: https://github.com/WebAssembly/wabt")
		}
		return "", nil, fmt.Errorf("wat2wasm %s: %w\n%s", watPath, err, out)
	}

	return wasmPath, cleanup, nil
}

// ------------------------------------------------------------------------------------------------
// ensureOutput ensures the target WAT file exists, compiling from Go source if present.
func ensureOutput(spec *Spec) error {
	// If input source exists, compile it to output WAT.
	if spec.Input != "" {
		if _, err := os.Stat(spec.Input); err == nil {
			return compileGoToWat(spec.Input, spec.Output)
		}
	}

	// Otherwise, verify the pre-compiled output WAT file already exists.
	if _, err := os.Stat(spec.Output); err != nil {
		return fmt.Errorf("neither input %q nor output %q found: %w", spec.Input, spec.Output, err)
	}

	return nil
}

// ------------------------------------------------------------------------------------------------
// compileGoToWat compiles a Go input source file to a WAT output file using wogo's internal parser.
func compileGoToWat(input, output string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("parser panic: %v", r)
		}
	}()

	content, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("read input %q: %w", input, err)
	}

	p := &parser.Parser{}
	wat := p.Parse(string(content))

	if dir := filepath.Dir(output); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir for output %q: %w", output, err)
		}
	}

	if err := os.WriteFile(output, []byte(wat), 0644); err != nil {
		return fmt.Errorf("write output %q: %w", output, err)
	}

	return nil
}

// ------------------------------------------------------------------------------------------------
// compileGoToWasm compiles a Go input source file to a WASM output file using wogo's internal compiler.
func compileGoToWasm(input, output string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("compiler panic: %v", r)
		}
	}()

	content, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("read input %q: %w", input, err)
	}

	c := compiler.NewCompiler()
	wasmBytes, err := c.Compile(string(content))
	if err != nil {
		return fmt.Errorf("compile %q: %w", input, err)
	}

	if dir := filepath.Dir(output); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir for output %q: %w", output, err)
		}
	}

	if err := os.WriteFile(output, wasmBytes, 0644); err != nil {
		return fmt.Errorf("write output %q: %w", output, err)
	}

	return nil
}
