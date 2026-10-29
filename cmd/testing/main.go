package main

import (
	"fmt"
	"os"
)

// ------------------------------------------------------------------------------------------------
type TestCase struct {
	Command   string   // what to call
	Signature string   // function signature (e.g. i32 i32 -> i32)
	Arguments []string // arguments to give it
	Expected  []string // what to expect back
}

// ------------------------------------------------------------------------------------------------
type Spec struct {
	Input  string // input file
	Output string // target output for WAT file
	Tests  []TestCase
}

// ------------------------------------------------------------------------------------------------
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error running test(s):", err)
		os.Exit(1)
	}
}
