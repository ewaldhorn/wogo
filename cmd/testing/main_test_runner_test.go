package main

import (
	"math"
	"strings"
	"testing"
)

// ------------------------------------------------------------------------------------------------
func TestVerifyResults(t *testing.T) {
	tests := []struct {
		name     string
		actual   []string
		expected []string
		wantErr  string
	}{
		{
			name:     "matches exactly",
			actual:   []string{"25"},
			expected: []string{"25"},
			wantErr:  "",
		},
		{
			name:     "multiple values match",
			actual:   []string{"1", "2", "3"},
			expected: []string{"1", "2", "3"},
			wantErr:  "",
		},
		{
			name:     "value mismatch",
			actual:   []string{"25"},
			expected: []string{"15"},
			wantErr:  "result 0: got 25, want 15",
		},
		{
			name:     "count mismatch",
			actual:   []string{"25", "10"},
			expected: []string{"25"},
			wantErr:  "got 2 result(s) [25 10], want 1",
		},
		{
			name:     "no values returned when expected",
			actual:   nil,
			expected: []string{"25"},
			wantErr:  "function returned no values, want 1",
		},
		{
			name:     "both empty",
			actual:   nil,
			expected: nil,
			wantErr:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyResults(tt.actual, tt.expected)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("got error %q, want containing %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

// ------------------------------------------------------------------------------------------------
func Test_isClose(t *testing.T) {
	tests := []struct {
		name   string
		a      float64
		b      float64
		absTol float64
		relTol float64
		want   bool
	}{
		{
			name:   "identical numbers",
			a:      1.0,
			b:      1.0,
			absTol: 0.0,
			relTol: 0.0,
			want:   true,
		},
		{
			name:   "identical zeroes",
			a:      0.0,
			b:      0.0,
			absTol: 0.0,
			relTol: 0.0,
			want:   true,
		},
		{
			name:   "within absolute tolerance",
			a:      10.0,
			b:      10.00005,
			absTol: 0.0001,
			relTol: 0.0,
			want:   true,
		},
		{
			name:   "exceeds absolute tolerance",
			a:      10.0,
			b:      10.0002,
			absTol: 0.0001,
			relTol: 0.0,
			want:   false,
		},
		{
			name:   "within relative tolerance",
			a:      1000.0,
			b:      1005.0,
			absTol: 0.0,
			relTol: 0.01,
			want:   true,
		},
		{
			name:   "exceeds relative tolerance",
			a:      1000.0,
			b:      1020.0,
			absTol: 0.0,
			relTol: 0.01,
			want:   false,
		},
		{
			name:   "negative numbers within tolerance",
			a:      -5.0,
			b:      -5.00005,
			absTol: 0.0001,
			relTol: 0.0,
			want:   true,
		},
		{
			name:   "negative numbers exceeds tolerance",
			a:      -5.0,
			b:      -5.0002,
			absTol: 0.0001,
			relTol: 0.0,
			want:   false,
		},
		{
			name:   "NaN values",
			a:      math.NaN(),
			b:      math.NaN(),
			absTol: 0.1,
			relTol: 0.1,
			want:   false,
		},
		{
			name:   "one NaN value",
			a:      math.NaN(),
			b:      1.0,
			absTol: 0.1,
			relTol: 0.1,
			want:   false,
		},
		{
			name:   "infinity identical",
			a:      math.Inf(1),
			b:      math.Inf(1),
			absTol: 0.1,
			relTol: 0.1,
			want:   true,
		},
		{
			name:   "infinity vs finite",
			a:      math.Inf(1),
			b:      100.0,
			absTol: 0.1,
			relTol: 0.1,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isClose(tt.a, tt.b, tt.absTol, tt.relTol)
			if got != tt.want {
				t.Errorf("isClose(%v, %v, %v, %v) = %v, want %v",
					tt.a, tt.b, tt.absTol, tt.relTol, got, tt.want)
			}
		})
	}
}

// ------------------------------------------------------------------------------------------------
func Test_equalNumbers(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
		want     bool
	}{
		{
			name:     "exact float strings",
			actual:   "3.14159",
			expected: "3.14159",
			want:     true,
		},
		{
			name:     "float strings with whitespace",
			actual:   "  3.14159  ",
			expected: "3.14159",
			want:     true,
		},
		{
			name:     "float close within tolerance",
			actual:   "3.14159265",
			expected: "3.14159266",
			want:     true,
		},
		{
			name:     "float difference exceeds tolerance",
			actual:   "3.14",
			expected: "3.15",
			want:     false,
		},
		{
			name:     "both NaN",
			actual:   "NaN",
			expected: "NaN",
			want:     true,
		},
		{
			name:     "one NaN and one number",
			actual:   "NaN",
			expected: "1.0",
			want:     false,
		},
		{
			name:     "actual not a float and not an int",
			actual:   "invalid",
			expected: "42",
			want:     false,
		},
		{
			name:     "expected not a float and not a uint",
			actual:   "42",
			expected: "invalid",
			want:     false,
		},
		{
			name:     "signed negative int matching uint32 representation (-1 and 4294967295)",
			actual:   "-1",
			expected: "4294967295",
			want:     true,
		},
		{
			name:     "signed negative int matching uint32 representation (-10 and 4294967286)",
			actual:   "-10",
			expected: "4294967286",
			want:     true,
		},
		{
			name:     "signed negative int not matching uint32 (-2 and 4294967295)",
			actual:   "-2",
			expected: "4294967295",
			want:     false,
		},
		{
			name:     "expected uint32 out of range for int32 matching",
			actual:   "-1",
			expected: "10000000000",
			want:     false,
		},
		{
			name:     "positive int64 matching uint64",
			actual:   "9000000000",
			expected: "9000000000",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalNumbers(tt.actual, tt.expected)
			if got != tt.want {
				t.Errorf("equalNumbers(%q, %q) = %v, want %v",
					tt.actual, tt.expected, got, tt.want)
			}
		})
	}
}

// ------------------------------------------------------------------------------------------------
func Test_matchesResult(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
		want     bool
	}{
		{
			name:     "exact strings",
			actual:   "hello",
			expected: "hello",
			want:     true,
		},
		{
			name:     "bool 0 and false",
			actual:   "0",
			expected: "false",
			want:     true,
		},
		{
			name:     "bool false and 0",
			actual:   "false",
			expected: "0",
			want:     true,
		},
		{
			name:     "bool 1 and true",
			actual:   "1",
			expected: "true",
			want:     true,
		},
		{
			name:     "bool true and 1",
			actual:   "true",
			expected: "1",
			want:     true,
		},
		{
			name:     "bool mismatch",
			actual:   "0",
			expected: "true",
			want:     false,
		},
		{
			name:     "delegates to equalNumbers",
			actual:   "180.0",
			expected: "180",
			want:     true,
		},
		{
			name:     "non-number mismatch",
			actual:   "abc",
			expected: "xyz",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesResult(tt.actual, tt.expected)
			if got != tt.want {
				t.Errorf("matchesResult(%q, %q) = %v, want %v",
					tt.actual, tt.expected, got, tt.want)
			}
		})
	}
}
