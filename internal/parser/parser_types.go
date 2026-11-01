package parser

import (
	"go/ast"
	"go/token"
)

// ------------------------------------------------------------------------------------------------
// typeName finds the name of a Go type expression. returns "int32" when unsure.
// TODO: Better error handling, "int32" is ok for now.
func typeName(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}

	return "int32"
}

// ------------------------------------------------------------------------------------------------
// toWasmType maps Go types to WebAssembly types.
// TODO: Again it defaults to i32 but I really should have an error handler here.
func toWasmType(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		switch ident.Name {
		case "int32", "uint32", "bool":
			return "i32"
		case "int64", "uint64":
			return "i64"
		case "float32":
			return "f32"
		case "float64":
			return "f64"
		}
	}

	return "i32"
}

// ------------------------------------------------------------------------------------------------
// wasmTypePrefix figures out if a type is integer or float and 32 or 64 bit.
// TODO: More erorr handling needed.
func wasmTypePrefix(exprType string) string {
	switch exprType {
	case "int64", "uint64":
		return "i64"
	case "float32":
		return "f32"
	case "float64":
		return "f64"
	default:
		return "i32"
	}
}

// ------------------------------------------------------------------------------------------------
// resolveExprType figures out what type is required for the expression
// TODO: I think the unary expressions need some work.
func (p *Parser) resolveExprType(expr ast.Expr, targetType string) string {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name == "true" || e.Name == "false" {
			return "bool"
		}
		if t, ok := p.localTypes[e.Name]; ok {
			return t
		}
		return targetType

	case *ast.BasicLit:
		if e.Kind == token.FLOAT {
			if targetType == "float32" {
				return "float32"
			}
			return "float64"
		}
		return targetType

	case *ast.BinaryExpr:
		tX := p.resolveExprType(e.X, targetType)
		if tX != "int32" && tX != "" {
			return tX
		}
		tY := p.resolveExprType(e.Y, targetType)
		if tY != "int32" && tY != "" {
			return tY
		}
		if tX != "" {
			return tX
		}
		return targetType

	case *ast.ParenExpr:
		return p.resolveExprType(e.X, targetType)

	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return "bool"
		}
		return p.resolveExprType(e.X, targetType)

	default:
		return targetType
	}
}
