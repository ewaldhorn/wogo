package ir

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

type mathConst struct {
	val float64
	raw string
}

var mathConstants = map[string]mathConst{
	"Pi":  {val: 3.141592653589793, raw: "3.141592653589793"},
	"E":   {val: 2.718281828459045, raw: "2.718281828459045"},
	"Phi": {val: 1.618033988749895, raw: "1.618033988749895"},
}

func (b *builder) zeroConstInstruction(wasmType Type) Instruction {
	switch wasmType {
	case TypeI64:
		return Instruction{Op: OpI64Const, I64: 0}
	case TypeF32:
		return Instruction{Op: OpF32Const, F32: 0}
	case TypeF64:
		return Instruction{Op: OpF64Const, F64: 0}
	default:
		return Instruction{Op: OpI32Const, I32: 0}
	}
}

func oneConstInstruction(goType string) (Instruction, error) {
	switch goType {
	case "int32", "int", "uint32", "uint":
		return Instruction{Op: OpI32Const, I32: 1}, nil
	case "int64", "uint64":
		return Instruction{Op: OpI64Const, I64: 1}, nil
	case "float32":
		return Instruction{Op: OpF32Const, F32: 1.0}, nil
	case "float64":
		return Instruction{Op: OpF64Const, F64: 1.0}, nil
	default:
		return Instruction{}, fmt.Errorf("unsupported type %s for ++/--", goType)
	}
}

func (b *builder) compileExpr(expr ast.Expr, targetType string) error {
	exprType := b.resolveExprType(expr, targetType)
	if exprType == "" {
		exprType = "int32"
	}

	switch e := expr.(type) {
	case *ast.Ident:
		return b.compileIdent(e, exprType)
	case *ast.BasicLit:
		return b.compileBasicLit(e, exprType)
	case *ast.BinaryExpr:
		return b.compileBinaryExpr(e, exprType)
	case *ast.UnaryExpr:
		return b.compileUnaryExpr(e, exprType)
	case *ast.ParenExpr:
		return b.compileExpr(e.X, exprType)
	case *ast.SelectorExpr:
		return b.compileSelectorExpr(e, exprType)
	default:
		return fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func (b *builder) compileIdent(e *ast.Ident, exprType string) error {
	if e.Name == "true" {
		b.emit(Instruction{Op: OpI32Const, I32: 1})
		return nil
	}
	if e.Name == "false" {
		b.emit(Instruction{Op: OpI32Const, I32: 0})
		return nil
	}
	if e.Name == "_" {
		return fmt.Errorf("cannot use _ as value")
	}

	binding, cDef, kind := b.lookupIdent(e.Name)
	switch kind {
	case identConst:
		targetType := exprType
		if cDef.explicitType != "" {
			targetType = cDef.explicitType
		}
		var compileErr error
		b.withConstEnv(cDef, func() {
			compileErr = b.compileExpr(cDef.expr, targetType)
		})
		return compileErr
	case identVar:
		b.emitLocalGet(binding)
		return nil
	default:
		return fmt.Errorf("unknown identifier: %s", e.Name)
	}
}

func (b *builder) compileBasicLit(e *ast.BasicLit, exprType string) error {
	switch e.Kind {
	case token.INT:
		val, err := strconv.ParseInt(e.Value, 0, 64)
		if err != nil {
			if uval, uerr := strconv.ParseUint(e.Value, 0, 64); uerr == nil {
				val = int64(uval)
			} else {
				return fmt.Errorf("parse integer literal %q: %w", e.Value, err)
			}
		}

		switch exprType {
		case "int64", "uint64":
			b.emit(Instruction{Op: OpI64Const, I64: val, Raw: e.Value})
		case "float32":
			b.emit(Instruction{Op: OpF32Const, F32: float32(val), Raw: e.Value})
		case "float64":
			b.emit(Instruction{Op: OpF64Const, F64: float64(val), Raw: e.Value})
		case "int32", "int", "uint32", "uint", "bool":
			b.emit(Instruction{Op: OpI32Const, I32: int32(val), Raw: e.Value})
		default:
			return fmt.Errorf("unsupported type %q for integer literal", exprType)
		}
		return nil

	case token.FLOAT:
		val, err := strconv.ParseFloat(e.Value, 64)
		if err != nil {
			return fmt.Errorf("parse float literal %q: %w", e.Value, err)
		}
		if exprType == "float32" {
			b.emit(Instruction{Op: OpF32Const, F32: float32(val), Raw: e.Value})
		} else {
			b.emit(Instruction{Op: OpF64Const, F64: val, Raw: e.Value})
		}
		return nil

	default:
		return fmt.Errorf("unsupported literal kind: %v", e.Kind)
	}
}

func (b *builder) compileBinaryExpr(e *ast.BinaryExpr, exprType string) error {
	opType := exprType
	if isComparison(e.Op) {
		var err error
		opType, err = b.resolveComparisonOperandType(e)
		if err != nil {
			return err
		}
	} else {
		cX := b.concreteExprType(e.X)
		cY := b.concreteExprType(e.Y)
		if cX != "" && cY != "" && cX != cY {
			return fmt.Errorf("invalid operation: %s (mismatched types %s and %s)", e.Op, cX, cY)
		}
		tX := b.resolveExprType(e.X, exprType)
		tY := b.resolveExprType(e.Y, exprType)
		if tX != "" && tY != "" && tX != tY {
			return fmt.Errorf("invalid operation: %s (mismatched types %s and %s)", e.Op, tX, tY)
		}
	}

	if err := b.compileExpr(e.X, opType); err != nil {
		return err
	}
	if err := b.compileExpr(e.Y, opType); err != nil {
		return err
	}

	opCode, err := binaryOpCode(e.Op, opType)
	if err != nil {
		return err
	}
	b.emit(Instruction{Op: opCode})
	return nil
}

func (b *builder) compileUnaryExpr(e *ast.UnaryExpr, exprType string) error {
	switch e.Op {
	case token.NOT:
		if err := b.compileExpr(e.X, "bool"); err != nil {
			return err
		}
		b.emit(Instruction{Op: OpI32Eqz})
		return nil

	case token.SUB:
		switch exprType {
		case "float32":
			if err := b.compileExpr(e.X, exprType); err != nil {
				return err
			}
			b.emit(Instruction{Op: OpF32Neg})
			return nil
		case "float64":
			if err := b.compileExpr(e.X, exprType); err != nil {
				return err
			}
			b.emit(Instruction{Op: OpF64Neg})
			return nil
		case "int64", "uint64":
			b.emit(Instruction{Op: OpI64Const, I64: 0})
			if err := b.compileExpr(e.X, exprType); err != nil {
				return err
			}
			b.emit(Instruction{Op: OpI64Sub})
			return nil
		default:
			b.emit(Instruction{Op: OpI32Const, I32: 0})
			if err := b.compileExpr(e.X, exprType); err != nil {
				return err
			}
			b.emit(Instruction{Op: OpI32Sub})
			return nil
		}

	default:
		return fmt.Errorf("unsupported unary operator: %s", e.Op)
	}
}

// TODO: Introduce an actual stdlib somehow; for now, provide common math constants.
func (b *builder) compileSelectorExpr(e *ast.SelectorExpr, exprType string) error {
	if x, ok := e.X.(*ast.Ident); ok && x.Name == "math" {
		if c, ok := mathConstants[e.Sel.Name]; ok {
			b.emit(Instruction{Op: OpF64Const, F64: c.val, Raw: c.raw})
			return nil
		}
	}
	return fmt.Errorf("unsupported selector expression: %v.%s", e.X, e.Sel.Name)
}
