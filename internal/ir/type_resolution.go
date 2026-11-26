package ir

import (
	"fmt"
	"go/ast"
	"go/token"
)

func isIntegerType(goType string) bool {
	switch goType {
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return true
	default:
		return false
	}
}

func isComparison(op token.Token) bool {
	switch op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}

// typeName renders a Go AST type expression as a type name. Only plain
// identifiers are supported; anything else returns an error rather than defaulting to int32.
func typeName(expr ast.Expr) (string, error) {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name, nil
	}
	return "", fmt.Errorf("unsupported type expression %T", expr)
}

func (b *builder) checkAssignable(expr ast.Expr, targetType string) error {
	resolved := b.resolveExprType(expr, targetType)
	if resolved == "" {
		return nil
	}
	if resolved != targetType {
		return fmt.Errorf("cannot use %s as %s in assignment", resolved, targetType)
	}
	if targetType == "bool" && b.resolveExprType(expr, "") == "" {
		return fmt.Errorf("cannot use untyped int as bool in assignment")
	}
	return nil
}

func (b *builder) resolveExprType(expr ast.Expr, targetType string) string {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name == "true" || e.Name == "false" {
			return "bool"
		}
		binding, cDef, kind := b.lookupIdent(e.Name)
		switch kind {
		case identVar:
			return binding.goType
		case identConst:
			if cDef.explicitType != "" {
				return cDef.explicitType
			}
			var resolved string
			b.withConstEnv(cDef, func() {
				resolved = b.resolveExprType(cDef.expr, targetType)
			})
			return resolved
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
		if isComparison(e.Op) {
			return "bool"
		}
		tX := b.resolveExprType(e.X, targetType)
		if tX != "int32" && tX != "" {
			return tX
		}
		tY := b.resolveExprType(e.Y, targetType)
		if tY != "int32" && tY != "" {
			return tY
		}
		if tX != "" {
			return tX
		}
		return targetType

	case *ast.ParenExpr:
		return b.resolveExprType(e.X, targetType)

	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return "bool"
		}
		return b.resolveExprType(e.X, targetType)

	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok && x.Name == "math" {
			if _, ok := mathConstants[e.Sel.Name]; ok {
				return "float64"
			}
		}
		return targetType

	default:
		return targetType
	}
}

// resolveComparisonOperandType determines the common operand type for a comparison.
func (b *builder) resolveComparisonOperandType(e *ast.BinaryExpr) (string, error) {
	cX := b.concreteExprType(e.X)
	cY := b.concreteExprType(e.Y)
	if cX != "" && cY != "" && cX != cY {
		return "", fmt.Errorf("invalid operation: %s (mismatched types %s and %s)", e.Op, cX, cY)
	}

	target := cX
	if target == "" {
		target = cY
	}

	tX := b.resolveExprType(e.X, target)
	tY := b.resolveExprType(e.Y, target)

	if tX != "" && tY != "" && tX != tY {
		return "", fmt.Errorf("invalid operation: %s (mismatched types %s and %s)", e.Op, tX, tY)
	}

	if tX != "" && tX != "int32" {
		return tX, nil
	}
	if tY != "" && tY != "int32" {
		return tY, nil
	}
	if tX != "" {
		return tX, nil
	}
	if tY != "" {
		return tY, nil
	}
	return "int32", nil
}

// concreteExprType returns the explicit or declared type of an expression if known,
// or empty string if it is an untyped literal or unconstrained expression.
func (b *builder) concreteExprType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name == "true" || e.Name == "false" {
			return "bool"
		}
		binding, cDef, kind := b.lookupIdent(e.Name)
		switch kind {
		case identVar:
			return binding.goType
		case identConst:
			if cDef.explicitType != "" {
				return cDef.explicitType
			}
		}
	case *ast.ParenExpr:
		return b.concreteExprType(e.X)
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return "bool"
		}
		return b.concreteExprType(e.X)
	case *ast.BinaryExpr:
		if isComparison(e.Op) {
			return "bool"
		}
		if c := b.concreteExprType(e.X); c != "" {
			return c
		}
		return b.concreteExprType(e.Y)
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok && x.Name == "math" {
			if _, ok := mathConstants[e.Sel.Name]; ok {
				return "float64"
			}
		}
	}
	return ""
}

var binaryOpTables = map[string]map[token.Token]OpCode{
	"bool": {
		token.EQL: OpI32Eq,
		token.NEQ: OpI32Ne,
	},
	"int64": {
		token.ADD: OpI64Add,
		token.SUB: OpI64Sub,
		token.MUL: OpI64Mul,
		token.QUO: OpI64DivS,
		token.REM: OpI64RemS,
		token.EQL: OpI64Eq,
		token.NEQ: OpI64Ne,
		token.LSS: OpI64LtS,
		token.LEQ: OpI64LeS,
		token.GTR: OpI64GtS,
		token.GEQ: OpI64GeS,
	},
	"uint64": {
		token.ADD: OpI64Add,
		token.SUB: OpI64Sub,
		token.MUL: OpI64Mul,
		token.QUO: OpI64DivU,
		token.REM: OpI64RemU,
		token.EQL: OpI64Eq,
		token.NEQ: OpI64Ne,
		token.LSS: OpI64LtU,
		token.LEQ: OpI64LeU,
		token.GTR: OpI64GtU,
		token.GEQ: OpI64GeU,
	},
	"int32": {
		token.ADD: OpI32Add,
		token.SUB: OpI32Sub,
		token.MUL: OpI32Mul,
		token.QUO: OpI32DivS,
		token.REM: OpI32RemS,
		token.EQL: OpI32Eq,
		token.NEQ: OpI32Ne,
		token.LSS: OpI32LtS,
		token.LEQ: OpI32LeS,
		token.GTR: OpI32GtS,
		token.GEQ: OpI32GeS,
	},
	"uint32": {
		token.ADD: OpI32Add,
		token.SUB: OpI32Sub,
		token.MUL: OpI32Mul,
		token.QUO: OpI32DivU,
		token.REM: OpI32RemU,
		token.EQL: OpI32Eq,
		token.NEQ: OpI32Ne,
		token.LSS: OpI32LtU,
		token.LEQ: OpI32LeU,
		token.GTR: OpI32GtU,
		token.GEQ: OpI32GeU,
	},
	"float32": {
		token.ADD: OpF32Add,
		token.SUB: OpF32Sub,
		token.MUL: OpF32Mul,
		token.QUO: OpF32Div,
		token.EQL: OpF32Eq,
		token.NEQ: OpF32Ne,
		token.LSS: OpF32Lt,
		token.LEQ: OpF32Le,
		token.GTR: OpF32Gt,
		token.GEQ: OpF32Ge,
	},
	"float64": {
		token.ADD: OpF64Add,
		token.SUB: OpF64Sub,
		token.MUL: OpF64Mul,
		token.QUO: OpF64Div,
		token.EQL: OpF64Eq,
		token.NEQ: OpF64Ne,
		token.LSS: OpF64Lt,
		token.LEQ: OpF64Le,
		token.GTR: OpF64Gt,
		token.GEQ: OpF64Ge,
	},
}

func normalizeOpType(exprType string) string {
	switch exprType {
	case "int":
		return "int32"
	case "uint":
		return "uint32"
	default:
		return exprType
	}
}

func binaryOpCode(op token.Token, exprType string) (OpCode, error) {
	table, ok := binaryOpTables[normalizeOpType(exprType)]
	if !ok {
		return 0, fmt.Errorf("unsupported type %q for binary operator %s", exprType, op)
	}
	opCode, ok := table[op]
	if !ok {
		return 0, fmt.Errorf("unsupported binary operator %s for type %s", op, exprType)
	}
	return opCode, nil
}
