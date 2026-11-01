package parser

import (
	"fmt"
	"go/ast"
	"go/token"
)

// ------------------------------------------------------------------------------------------------
// parseExpression parses an expression using the active return type.
func (p *Parser) parseExpression(expr ast.Expr) error {
	return p.parseExprWithTarget(expr, p.returnType)
}

// ------------------------------------------------------------------------------------------------
// parseExprWithTarget figures out which type of expression is needed to be parsed and does so.
func (p *Parser) parseExprWithTarget(expr ast.Expr, targetType string) error {
	exprType := p.resolveExprType(expr, targetType)
	if exprType == "" {
		exprType = "int32"
	}

	switch e := expr.(type) {
	case *ast.Ident:
		return p.parseIdent(e)
	case *ast.BasicLit:
		return p.parseBasicLit(e, exprType)
	case *ast.BinaryExpr:
		return p.parseBinaryExpr(e, exprType)
	case *ast.UnaryExpr:
		return p.parseUnaryExpr(e, exprType)
	case *ast.ParenExpr:
		return p.parseExprWithTarget(e.X, exprType)
	default:
		return fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// ------------------------------------------------------------------------------------------------
// parseIdent tries to parse an identifier.
// TODO: Error handling is basically not there.
func (p *Parser) parseIdent(e *ast.Ident) error {
	if e.Name == "true" {
		p.emit("i32.const 1")
		return nil
	}
	if e.Name == "false" {
		p.emit("i32.const 0")
		return nil
	}

	wasmName, ok := p.locals[e.Name]
	if !ok {
		return fmt.Errorf("unknown identifier: %s", e.Name)
	}
	p.emit("local.get %s", wasmName)

	return nil
}

// ------------------------------------------------------------------------------------------------
// parseBasicLit tries to parse literals.
// TODO: This might need tweaking when I finally get string literals. Alos not sure about cosntants.
func (p *Parser) parseBasicLit(e *ast.BasicLit, exprType string) error {
	switch e.Kind {
	case token.INT:
		p.emit("%s.const %s", wasmTypePrefix(exprType), e.Value)
		return nil

	case token.FLOAT:
		prefix := "f64"
		if exprType == "float32" {
			prefix = "f32"
		}
		p.emit("%s.const %s", prefix, e.Value)
		return nil

	default:
		return fmt.Errorf("unsupported literal kind: %v", e.Kind)
	}
}

// ------------------------------------------------------------------------------------------------
// parseBinaryExpr parses a binary expression like { a + b }
func (p *Parser) parseBinaryExpr(e *ast.BinaryExpr, exprType string) error {
	if err := p.parseExprWithTarget(e.X, exprType); err != nil {
		return err
	}
	if err := p.parseExprWithTarget(e.Y, exprType); err != nil {
		return err
	}

	inst, err := binaryOpInstruction(e.Op, exprType)
	if err != nil {
		return err
	}
	p.emit("%s", inst)
	return nil
}

// ------------------------------------------------------------------------------------------------
// binaryOpInstruction figures out which instruction is appropriate for the token.
// for example, MUL token (*) needs .mul
func binaryOpInstruction(op token.Token, exprType string) (string, error) {
	prefix := wasmTypePrefix(exprType)
	switch op {
	case token.ADD:
		return prefix + ".add", nil
	case token.SUB:
		return prefix + ".sub", nil
	case token.MUL:
		return prefix + ".mul", nil
	case token.QUO:
		switch exprType {
		case "uint32":
			return "i32.div_u", nil
		case "uint64":
			return "i64.div_u", nil
		case "int64":
			return "i64.div_s", nil
		case "float32":
			return "f32.div", nil
		case "float64":
			return "f64.div", nil
		default:
			return "i32.div_s", nil
		}
	case token.REM:
		switch exprType {
		case "uint32":
			return "i32.rem_u", nil
		case "uint64":
			return "i64.rem_u", nil
		case "int64":
			return "i64.rem_s", nil
		default:
			return "i32.rem_s", nil
		}
	default:
		return "", fmt.Errorf("unsupported operator: %s", op)
	}
}

// ------------------------------------------------------------------------------------------------
// parseUnaryExpr handles single argument expressions, like !,+,- etc.
func (p *Parser) parseUnaryExpr(e *ast.UnaryExpr, exprType string) error {
	switch e.Op {
	case token.ADD:
		return p.parseExprWithTarget(e.X, exprType)

	case token.NOT:
		if err := p.parseExprWithTarget(e.X, "bool"); err != nil {
			return err
		}
		p.emit("i32.eqz")
		return nil

	case token.SUB:
		if exprType == "float32" || exprType == "float64" {
			if err := p.parseExprWithTarget(e.X, exprType); err != nil {
				return err
			}
			p.emit("%s.neg", wasmTypePrefix(exprType))
			return nil
		}
		prefix := wasmTypePrefix(exprType)
		p.emit("%s.const 0", prefix)
		if err := p.parseExprWithTarget(e.X, exprType); err != nil {
			return err
		}
		p.emit("%s.sub", prefix)
		return nil

	default:
		return fmt.Errorf("unsupported unary operator: %s", e.Op)
	}
}
