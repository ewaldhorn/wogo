package main

import (
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
