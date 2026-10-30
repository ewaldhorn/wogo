package compiler

import "go/ast"

// ------------------------------------------------------------------------------------------------
// Number types in WebAssembly.
// See https://webassembly.github.io/spec/core/binary/types.html
const (
	TypeI32 byte = 0x7F
	TypeI64 byte = 0x7E
	TypeF32 byte = 0x7D
	TypeF64 byte = 0x7C
)

// ------------------------------------------------------------------------------------------------
// toWasmValueType converts an ast.Expr Go type into a WebAssembly value type byte code.
func toWasmValueType(expr ast.Expr) byte {
	if ident, ok := expr.(*ast.Ident); ok {
		switch ident.Name {
		case "int32", "uint32", "bool":
			return TypeI32
		case "int64", "uint64":
			return TypeI64
		case "float32":
			return TypeF32
		case "float64":
			return TypeF64
		}
	}
	return TypeI32 // when in doubt, see if i32 will work
}
