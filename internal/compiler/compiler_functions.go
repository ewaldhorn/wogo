package compiler

import (
	"bytes"
	"go/ast"
)

// ------------------------------------------------------------------------------------------------
// what params and what return types does this function have
type funcSignature struct {
	params  []byte
	results []byte
}

// ------------------------------------------------------------------------------------------------
// which function is this, what type is it (the signature) and what's its compiled state
type compiledFunction struct {
	name      string
	typeIndex uint32
	body      []byte
}

// ------------------------------------------------------------------------------------------------
// Attempts to compile a function, by compiling the statements which in turn do the expressions.
func (c *Compiler) compileFunc(fn *ast.FuncDecl) (funcSignature, []byte, error) {
	c.locals = make(map[string]uint32)
	c.localTypes = make(map[string]string)
	c.returnType = "int32"

	var sig funcSignature
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			goType := typeName(field.Type)
			wasmType := toWasmValueType(field.Type)
			for _, name := range field.Names {
				c.locals[name.Name] = uint32(len(c.locals))
				c.localTypes[name.Name] = goType
				sig.params = append(sig.params, wasmType)
			}
		}
	}

	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			c.returnType = typeName(field.Type)
			sig.results = append(sig.results, toWasmValueType(field.Type))
		}
	}

	var code bytes.Buffer
	for _, stmt := range fn.Body.List {
		if err := c.compileStatement(&code, stmt); err != nil {
			return sig, nil, err
		}
	}
	code.WriteByte(OpEnd)

	// In the code section, each function has:
	// 1. vector of local variable declarations (0 if only parameters are used)
	// 2. bytecode instructions terminated by OpEnd (0x0B)
	var body bytes.Buffer
	putUvarint(&body, 0) // 0 local variable declarations for now, I think this is a problem for later
	body.Write(code.Bytes())

	return sig, body.Bytes(), nil
}

// ------------------------------------------------------------------------------------------------
// typeName figures out the type name for an expression, as a string
func typeName(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "int32"
}
