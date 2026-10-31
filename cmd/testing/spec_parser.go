package main

import (
	"errors"
	"fmt"
	"strings"
)

// ------------------------------------------------------------------------------------------------
type SpecParser struct {
	spec    Spec
	current *TestCase
	lineNo  int

	// we need all of these or the test case is incomplete
	hasSignature bool
	hasArguments bool
	hasExpected  bool
}

// ------------------------------------------------------------------------------------------------
func (p *SpecParser) parseKeyValue(key, value string) error {
	switch key {
	case "input", "output":
		return p.parseHeader(key, value)
	case "command", "signature", "arguments", "expected":
		return p.parseTestField(key, value)
	default:
		return fmt.Errorf("line %d: unknown key %q", p.lineNo, key)
	}
}

// ------------------------------------------------------------------------------------------------
func (p *SpecParser) parseHeader(key, value string) error {
	if p.current != nil || len(p.spec.Tests) > 0 {
		return fmt.Errorf("line %d: %s must appear before tests", p.lineNo, key)
	}

	switch key {
	case "input":
		if p.spec.Input != "" {
			return fmt.Errorf("line %d: duplicate input", p.lineNo)
		}
		p.spec.Input = value

	case "output":
		if p.spec.Output != "" {
			return fmt.Errorf("line %d: duplicate output", p.lineNo)
		}
		p.spec.Output = value
	}

	return nil
}

// ------------------------------------------------------------------------------------------------
func (p *SpecParser) parseTestField(key, value string) error {
	if key == "command" {
		if err := p.flush(); err != nil {
			return err
		}
		if value == "" {
			return fmt.Errorf("line %d: empty command", p.lineNo)
		}
		p.current = &TestCase{Command: value}
		return nil
	}

	if p.current == nil {
		return fmt.Errorf("line %d: %s without command", p.lineNo, key)
	}

	switch key {
	case "signature":
		if p.hasSignature {
			return fmt.Errorf("line %d: duplicate signature", p.lineNo)
		}
		p.current.Signature = value
		p.hasSignature = true

	case "arguments":
		if p.hasArguments {
			return fmt.Errorf("line %d: duplicate arguments", p.lineNo)
		}
		p.current.Arguments = strings.Fields(value)
		p.hasArguments = true

	case "expected":
		if p.hasExpected {
			return fmt.Errorf("line %d: duplicate expected", p.lineNo)
		}
		p.current.Expected = strings.Fields(value)
		p.hasExpected = true
	}

	return nil
}

// ------------------------------------------------------------------------------------------------
// If the current spec is alright, append it to the list of tests, then reset the current spec.
func (p *SpecParser) flush() error {
	if p.current == nil {
		return nil
	}
	if !p.hasArguments {
		if p.hasNoArguments() {
			p.current.Arguments = nil
		} else {
			return fmt.Errorf("test %q: missing arguments", p.current.Command)
		}
	}
	if !p.hasExpected {
		return fmt.Errorf("test %q: missing expected", p.current.Command)
	}

	p.spec.Tests = append(p.spec.Tests, *p.current)
	p.current = nil
	p.hasSignature = false
	p.hasArguments = false
	p.hasExpected = false

	return nil
}

func (p *SpecParser) hasNoArguments() bool {
	if !p.hasSignature || p.current == nil {
		return false
	}
	parts := strings.Split(p.current.Signature, "->")
	if len(parts) < 2 {
		return false
	}
	params := strings.Fields(parts[0])
	return len(params) == 0
}

// ------------------------------------------------------------------------------------------------
func (p *SpecParser) validate() error {
	if p.spec.Input == "" {
		return errors.New("missing input")
	}
	if p.spec.Output == "" {
		return errors.New("missing output")
	}
	if len(p.spec.Tests) == 0 {
		return errors.New("no test cases found")
	}

	return nil
}
