package main

import (
	"strings"
	"testing"
)

// ------------------------------------------------------------------------------------------------
func TestParseSpec_Valid(t *testing.T) {
	input := `
input: testdata/functions/sum_two_ints.go
output: tmp/sum.wat

command: Sum
signature: i32 i32 -> i32
arguments: 10 15
expected: 25

command: Sum
signature: i32 i32 -> i32
arguments: 0 0
expected: 0
`
	spec, err := ParseSpec(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.Input != "testdata/functions/sum_two_ints.go" {
		t.Errorf("got input %q, want %q", spec.Input, "testdata/functions/sum_two_ints.go")
	}
	if spec.Output != "tmp/sum.wat" {
		t.Errorf("got output %q, want %q", spec.Output, "tmp/sum.wat")
	}
	if len(spec.Tests) != 2 {
		t.Fatalf("got %d tests, want 2", len(spec.Tests))
	}
	if spec.Tests[0].Command != "Sum" || spec.Tests[0].Signature != "i32 i32 -> i32" {
		t.Errorf("unexpected test case 0: %+v", spec.Tests[0])
	}
}

// ------------------------------------------------------------------------------------------------
func TestParseSpec_Errors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "missing input",
			input:   "output: foo.wat\ncommand: Sum\narguments: 1\nexpected: 1",
			wantErr: "missing input",
		},
		{
			name:    "missing output",
			input:   "input: foo.go\ncommand: Sum\narguments: 1\nexpected: 1",
			wantErr: "missing output",
		},
		{
			name:    "duplicate input",
			input:   "input: a.go\ninput: b.go\noutput: out.wat",
			wantErr: "duplicate input",
		},
		{
			name:    "duplicate output",
			input:   "input: a.go\noutput: a.wat\noutput: b.wat",
			wantErr: "duplicate output",
		},
		{
			name:    "input after tests",
			input:   "output: a.wat\ncommand: Sum\narguments: 1\nexpected: 1\ninput: a.go",
			wantErr: "input must appear before tests",
		},
		{
			name:    "unknown key",
			input:   "input: a.go\noutput: a.wat\nfoo: bar",
			wantErr: "unknown key \"foo\"",
		},
		{
			name:    "no test cases",
			input:   "input: a.go\noutput: a.wat",
			wantErr: "no test cases found",
		},
		{
			name:    "missing arguments",
			input:   "input: a.go\noutput: a.wat\ncommand: Sum\nexpected: 1",
			wantErr: "missing arguments",
		},
		{
			name:    "missing expected",
			input:   "input: a.go\noutput: a.wat\ncommand: Sum\narguments: 1",
			wantErr: "missing expected",
		},
		{
			name:    "duplicate arguments",
			input:   "input: a.go\noutput: a.wat\ncommand: Sum\narguments: 1\narguments: 2\nexpected: 3",
			wantErr: "duplicate arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSpec(strings.NewReader(tt.input))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("got error %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}
