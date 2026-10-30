package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

// ------------------------------------------------------------------------------------------------
// Attempts to compile an expression. Lots to do here still.
func (c *Compiler) compileExpression(buf *bytes.Buffer, expr ast.Expr) error {
	switch e := expr.(type) {
	case *ast.Ident:
		idx, ok := c.locals[e.Name]
		if !ok {
			return fmt.Errorf("unknown identifier: %s", e.Name)
		}
		buf.WriteByte(OpLocalGet)
		putUvarint(buf, idx)
		return nil

	case *ast.BasicLit:
		if e.Kind == token.INT {
			val, err := strconv.ParseInt(e.Value, 0, 32)
			if err != nil {
				return fmt.Errorf("parse integer literal %q: %w", e.Value, err)
			}
			buf.WriteByte(OpI32Const)
			putVarint(buf, int32(val))
			return nil
		}
		return fmt.Errorf("unsupported literal kind: %v", e.Kind)

	case *ast.BinaryExpr:
		if err := c.compileExpression(buf, e.X); err != nil {
			return err
		}
		if err := c.compileExpression(buf, e.Y); err != nil {
			return err
		}

		switch e.Op {
		case token.ADD:
			buf.WriteByte(OpI32Add)
		case token.SUB:
			buf.WriteByte(OpI32Sub)
		case token.MUL:
			buf.WriteByte(OpI32Mul)
		case token.QUO:
			buf.WriteByte(OpI32DivS)
		case token.REM:
			buf.WriteByte(OpI32RemS)
		default:
			return fmt.Errorf("unsupported operator: %s", e.Op)
		}
		return nil

	case *ast.ParenExpr:
		return c.compileExpression(buf, e.X)

	default:
		return fmt.Errorf("unsupported expression type: %T", expr)
	}
}
