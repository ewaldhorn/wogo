package compiler

import (
	"bytes"
	"fmt"
	"wogo/internal/ir"
)

// ------------------------------------------------------------------------------------------------
// funcSignature holds parameters and results type bytes for a function type in WASM.
type funcSignature struct {
	params  []byte
	results []byte
}

// ------------------------------------------------------------------------------------------------
// compiledFunction holds the name, typeIndex, and encoded body of a function.
type compiledFunction struct {
	name      string
	typeIndex uint32
	body      []byte
}

// ------------------------------------------------------------------------------------------------
// Compiler translates Go source code into WebAssembly binary bytecode (.wasm).
type Compiler struct{}

// ------------------------------------------------------------------------------------------------
func NewCompiler() *Compiler {
	return &Compiler{}
}

// ------------------------------------------------------------------------------------------------
// Compile compiles Go source code into WebAssembly binary format bytes using the IR frontend.
func (c *Compiler) Compile(src string) ([]byte, error) {
	mod, err := ir.BuildModule(src)
	if err != nil {
		return nil, err
	}

	var signatures []funcSignature
	var functions []compiledFunction

	for _, fn := range mod.Functions {
		sig := funcSignature{}
		for _, p := range fn.Params {
			sig.params = append(sig.params, byte(p.Type))
		}
		for _, r := range fn.Results {
			sig.results = append(sig.results, byte(r))
		}

		var code bytes.Buffer
		for _, inst := range fn.Instructions {
			ir.EncodeInstruction(&code, inst)
		}
		code.WriteByte(byte(ir.OpEnd))

		var body bytes.Buffer
		putUvarint(&body, 0) // 0 local variable declarations for now
		body.Write(code.Bytes())

		typeIndex := uint32(len(signatures))
		signatures = append(signatures, sig)
		functions = append(functions, compiledFunction{
			name:      fn.Name,
			typeIndex: typeIndex,
			body:      body.Bytes(),
		})
	}

	return c.assembleModule(signatures, functions)
}

// ------------------------------------------------------------------------------------------------
// assembleModule puts together a wasm module with the header and all the sections.
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
