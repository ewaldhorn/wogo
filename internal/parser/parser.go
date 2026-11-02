package parser

import (
	"bytes"
	"fmt"
	"strings"
	"wogo/internal/ir"
)

// ------------------------------------------------------------------------------------------------
type Parser struct {
	buf bytes.Buffer
}

// ------------------------------------------------------------------------------------------------
// Parse translates Go source code into WebAssembly text format (WAT).
// TODO: Spacing is a bit brittle I think
func (p *Parser) Parse(src string) (string, error) {
	mod, err := ir.BuildModule(src)
	if err != nil {
		return "", err
	}

	p.buf.Reset()
	p.emit("(module")

	for _, fn := range mod.Functions {
		var params []string
		for _, param := range fn.Params {
			params = append(params, fmt.Sprintf("(param $%s %s)", param.Name, param.Type))
		}
		var results []string
		for _, result := range fn.Results {
			results = append(results, fmt.Sprintf("(result %s)", result))
		}

		paramStr := strings.Join(params, " ")
		resultStr := strings.Join(results, " ")
		if paramStr != "" {
			paramStr = " " + paramStr
		}
		if resultStr != "" {
			resultStr = " " + resultStr
		}

		p.emit("  (func $%s (export %q)%s%s", fn.Name, fn.Name, paramStr, resultStr)

		// local variables first
		for _, loc := range fn.Locals {
			p.emit("    (local $%s %s)", loc.Name, loc.Type)
		}

		// now the actual instructions
		for _, inst := range fn.Instructions {
			wat := inst.WatString()
			if wat != "" {
				p.emit("    %s", wat)
			}
		}
		p.emit("  )")
	}

	p.emit(")")
	return p.buf.String(), nil
}

// ------------------------------------------------------------------------------------------------
// emit prints an indented or formatted string to the parser buffer.
func (p *Parser) emit(format string, args ...any) {
	fmt.Fprintf(&p.buf, format, args...)
	p.buf.WriteByte('\n')
}
