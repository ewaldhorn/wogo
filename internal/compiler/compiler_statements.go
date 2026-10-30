package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
)

// ------------------------------------------------------------------------------------------------
// Takes a statement, breats it down into expressions and tries to compile it.
func (c *Compiler) compileStatement(buf *bytes.Buffer, stmt ast.Stmt) error {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		for _, expr := range s.Results {
			if err := c.compileExpression(buf, expr); err != nil {
				return err
			}
		}
		buf.WriteByte(OpReturn)
		return nil
	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}
