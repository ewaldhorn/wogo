package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ------------------------------------------------------------------------------------------------
// ParseSpec parses the test specification format:
//
//	input: testdata/functions/sum_two_ints.go
//	output: sum_test.wat
//
//	command: Sum
//	signature: i32 i32 -> i32
//	arguments: 10 15
//	expected: 15
//
// The input/output lines appear once before any test cases.
// Every "command:" line starts a new test case.
func ParseSpec(r io.Reader) (*Spec, error) {
	p := &SpecParser{}
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		p.lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("line %d: expected \"key: value\"", p.lineNo)
		}

		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)

		if err := p.parseKeyValue(key, value); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("line %d: scanner error: %w", p.lineNo, err)
	}

	if err := p.flush(); err != nil {
		return nil, err
	}

	if err := p.validate(); err != nil {
		return nil, err
	}

	return &p.spec, nil
}
