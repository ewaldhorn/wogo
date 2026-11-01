package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
)

// ------------------------------------------------------------------------------------------------
// Compiler translates Go source code into WebAssembly binary bytecode (.wasm).
type Compiler struct {
	locals     map[string]uint32
	localTypes map[string]string
	returnType string
}

// ------------------------------------------------------------------------------------------------
func NewCompiler() *Compiler {
	return &Compiler{
		locals:     make(map[string]uint32),
		localTypes: make(map[string]string),
	}
}

// ------------------------------------------------------------------------------------------------
// Compile compiles Go source code into WebAssembly binary format bytes.
func (c *Compiler) Compile(src string) ([]byte, error) {
	fset := token.NewFileSet()
	node, err := goparser.ParseFile(fset, "calc.go", src, 0)
	if err != nil {
		return nil, fmt.Errorf("parse go source: %w", err)
	}

	var signatures []funcSignature
	var functions []compiledFunction

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		sig, body, err := c.compileFunc(fn)
		if err != nil {
			return nil, fmt.Errorf("compile function %s: %w", fn.Name.Name, err)
		}

		typeIndex := uint32(len(signatures))
		signatures = append(signatures, sig)
		functions = append(functions, compiledFunction{
			name:      fn.Name.Name,
			typeIndex: typeIndex,
			body:      body,
		})
	}

	return c.assembleModule(signatures, functions)
}

// ------------------------------------------------------------------------------------------------
// assembleModule puts together a wasm module with the header and all the sections
// TODO: Import section still baffles me.
func (c *Compiler) assembleModule(signatures []funcSignature, functions []compiledFunction) ([]byte, error) {
	var module bytes.Buffer

	// 1. Header
	if err := writeStandardWasmBinaryHeader(&module); err != nil {
		return nil, fmt.Errorf("writing wasm binary header: %w", err)
	}

	// 2. Sections
	writeTypeSection(&module, signatures)
	writeFunctionSection(&module, functions)
	writeExportSection(&module, functions)
	writeCodeSection(&module, functions)

	return module.Bytes(), nil
}
