package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

// ------------------------------------------------------------------------------------------------
// compileExpression compiles an expression using the current function's return type as target.
func (c *Compiler) compileExpression(buf *bytes.Buffer, expr ast.Expr) error {
	return c.compileExprWithTarget(buf, expr, c.returnType)
}

// ------------------------------------------------------------------------------------------------
// resolveExprType figures out the Go type for an expression given a target type.
func (c *Compiler) resolveExprType(expr ast.Expr, targetType string) string {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name == "true" || e.Name == "false" {
			return "bool"
		}
		if t, ok := c.localTypes[e.Name]; ok {
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
		tX := c.resolveExprType(e.X, targetType)
		if tX != "int32" && tX != "" {
			return tX
		}
		tY := c.resolveExprType(e.Y, targetType)
		if tY != "int32" && tY != "" {
			return tY
		}
		if tX != "" {
			return tX
		}
		return targetType

	case *ast.ParenExpr:
		return c.resolveExprType(e.X, targetType)

	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return "bool"
		}
		return c.resolveExprType(e.X, targetType)

	default:
		return targetType
	}
}

// ------------------------------------------------------------------------------------------------
// compileExprWithTarget picks the right compile type for an expression.
func (c *Compiler) compileExprWithTarget(buf *bytes.Buffer, expr ast.Expr, targetType string) error {
	exprType := c.resolveExprType(expr, targetType)
	if exprType == "" {
		exprType = "int32"
	}

	switch e := expr.(type) {
	case *ast.Ident:
		return c.compileIdent(buf, e)
	case *ast.BasicLit:
		return c.compileBasicLit(buf, e, exprType)
	case *ast.BinaryExpr:
		return c.compileBinaryExpr(buf, e, exprType)
	case *ast.UnaryExpr:
		return c.compileUnaryExpr(buf, e, exprType)
	case *ast.ParenExpr:
		return c.compileExprWithTarget(buf, e.X, exprType)
	default:
		return fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// ------------------------------------------------------------------------------------------------
// compileIdent compiles an identifier and figures out if it's a constant, local or something else.
func (c *Compiler) compileIdent(buf *bytes.Buffer, e *ast.Ident) error {
	if e.Name == "true" {
		buf.WriteByte(OpI32Const)
		putVarint(buf, 1)
		return nil
	}
	if e.Name == "false" {
		buf.WriteByte(OpI32Const)
		putVarint(buf, 0)
		return nil
	}

	idx, ok := c.locals[e.Name]
	if !ok {
		return fmt.Errorf("unknown identifier: %s", e.Name)
	}
	buf.WriteByte(OpLocalGet)
	putUvarint(buf, idx)
	return nil
}

// ------------------------------------------------------------------------------------------------
// compileBasicLit compiles an integer or float literal, creating a constant for it.
// TODO: I don't know if this is right for everything. It seems to work, but... Yeah.
func (c *Compiler) compileBasicLit(buf *bytes.Buffer, e *ast.BasicLit, exprType string) error {
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
			buf.WriteByte(OpI64Const)
			putVarint(buf, val)
		case "float32":
			buf.WriteByte(OpF32Const)
			putFloat32(buf, float32(val))
		case "float64":
			buf.WriteByte(OpF64Const)
			putFloat64(buf, float64(val))
		default:
			buf.WriteByte(OpI32Const)
			putVarint(buf, val)
		}
		return nil

	case token.FLOAT:
		val, err := strconv.ParseFloat(e.Value, 64)
		if err != nil {
			return fmt.Errorf("parse float literal %q: %w", e.Value, err)
		}
		if exprType == "float32" {
			buf.WriteByte(OpF32Const)
			putFloat32(buf, float32(val))
		} else {
			buf.WriteByte(OpF64Const)
			putFloat64(buf, val)
		}
		return nil

	default:
		return fmt.Errorf("unsupported literal kind: %v", e.Kind)
	}
}

// ------------------------------------------------------------------------------------------------
// compileBinaryExpr compiles a binary expression.
func (c *Compiler) compileBinaryExpr(buf *bytes.Buffer, e *ast.BinaryExpr, exprType string) error {
	if err := c.compileExprWithTarget(buf, e.X, exprType); err != nil {
		return err
	}
	if err := c.compileExprWithTarget(buf, e.Y, exprType); err != nil {
		return err
	}

	op, err := binaryOpCode(e.Op, exprType)
	if err != nil {
		return err
	}
	buf.WriteByte(op)
	return nil
}

// ------------------------------------------------------------------------------------------------
// binaryOpCode gets the wasm op code for an expression, errors if it doesn't support it yet.
// TODO: This is going to get messy...
func binaryOpCode(op token.Token, exprType string) (byte, error) {
	switch op {
	case token.ADD:
		switch exprType {
		case "int64", "uint64":
			return OpI64Add, nil
		case "float32":
			return OpF32Add, nil
		case "float64":
			return OpF64Add, nil
		default:
			return OpI32Add, nil
		}
	case token.SUB:
		switch exprType {
		case "int64", "uint64":
			return OpI64Sub, nil
		case "float32":
			return OpF32Sub, nil
		case "float64":
			return OpF64Sub, nil
		default:
			return OpI32Sub, nil
		}
	case token.MUL:
		switch exprType {
		case "int64", "uint64":
			return OpI64Mul, nil
		case "float32":
			return OpF32Mul, nil
		case "float64":
			return OpF64Mul, nil
		default:
			return OpI32Mul, nil
		}
	case token.QUO:
		switch exprType {
		case "uint32":
			return OpI32DivU, nil
		case "int64":
			return OpI64DivS, nil
		case "uint64":
			return OpI64DivU, nil
		case "float32":
			return OpF32Div, nil
		case "float64":
			return OpF64Div, nil
		default:
			return OpI32DivS, nil
		}
	case token.REM:
		switch exprType {
		case "uint32":
			return OpI32RemU, nil
		case "int64":
			return OpI64RemS, nil
		case "uint64":
			return OpI64RemU, nil
		default:
			return OpI32RemS, nil
		}
	default:
		return 0, fmt.Errorf("unsupported operator: %s", op)
	}
}

// ------------------------------------------------------------------------------------------------
// compileUnaryExpr compiles a unary expression
func (c *Compiler) compileUnaryExpr(buf *bytes.Buffer, e *ast.UnaryExpr, exprType string) error {
	switch e.Op {
	case token.ADD:
		return c.compileExprWithTarget(buf, e.X, exprType)

	case token.NOT:
		if err := c.compileExprWithTarget(buf, e.X, "bool"); err != nil {
			return err
		}
		buf.WriteByte(OpI32Eqz)
		return nil

	case token.SUB:
		switch exprType {
		case "float32":
			if err := c.compileExprWithTarget(buf, e.X, "float32"); err != nil {
				return err
			}
			buf.WriteByte(OpF32Neg)
		case "float64":
			if err := c.compileExprWithTarget(buf, e.X, "float64"); err != nil {
				return err
			}
			buf.WriteByte(OpF64Neg)
		case "int64", "uint64":
			buf.WriteByte(OpI64Const)
			putVarint(buf, 0)
			if err := c.compileExprWithTarget(buf, e.X, exprType); err != nil {
				return err
			}
			buf.WriteByte(OpI64Sub)
		default:
			buf.WriteByte(OpI32Const)
			putVarint(buf, 0)
			if err := c.compileExprWithTarget(buf, e.X, exprType); err != nil {
				return err
			}
			buf.WriteByte(OpI32Sub)
		}
		return nil

	default:
		return fmt.Errorf("unsupported unary operator: %s", e.Op)
	}
}
