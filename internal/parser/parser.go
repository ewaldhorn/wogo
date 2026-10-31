package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strings"
)

// ------------------------------------------------------------------------------------------------
type Parser struct {
	buf    bytes.Buffer
	indent int
	locals map[string]string // Map Go to Wasm local/param
}

// ------------------------------------------------------------------------------------------------
func (p *Parser) Parse(src string) string {
	fset := token.NewFileSet()
	node, err := goparser.ParseFile(fset, "calc.go", src, 0)
	if err != nil {
		panic(err)
	}

	p.emit("(module")
	p.indent++

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if err := p.ParseFunc(fn); err != nil {
				panic(err)
			}
		}
	}

	p.indent--
	p.emit(")")

	return p.buf.String()
}

// ------------------------------------------------------------------------------------------------
// Helper to indent to the current indention level
func (p *Parser) emit(format string, args ...any) {
	p.buf.WriteString(strings.Repeat("  ", p.indent))
	fmt.Fprintf(&p.buf, format, args...)
	p.buf.WriteByte('\n')
}

// ------------------------------------------------------------------------------------------------
// Parse ast.FuncDecl into WAT syntax
func (p *Parser) ParseFunc(funcDecl *ast.FuncDecl) error {
	p.locals = make(map[string]string)
	functionName := funcDecl.Name.Name

	var params []string
	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			wasmType := toWasmType(field.Type)
			for _, name := range field.Names {
				p.locals[name.Name] = "$" + name.Name
				params = append(params, fmt.Sprintf("(param $%s %s)", name.Name, wasmType))
			}
		}
	}

	var results []string
	if funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			results = append(results, fmt.Sprintf("(result %s)", toWasmType(field.Type)))
		}
	}

	paramStr := strings.Join(params, " ")
	resultStr := strings.Join(results, " ")

	p.emit("(func $%s (export %q) %s %s", functionName, functionName, paramStr, resultStr)
	p.indent++

	for _, stmt := range funcDecl.Body.List {
		if err := p.parseStatement(stmt); err != nil {
			return err
		}
	}

	p.indent--
	p.emit(")")
	return nil
}

// ------------------------------------------------------------------------------------------------
func (p *Parser) parseStatement(stmt ast.Stmt) error {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		for _, expr := range s.Results {
			if err := p.parseExpression(expr); err != nil {
				return err
			}
		}
		p.emit("return")
		return nil
	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// ------------------------------------------------------------------------------------------------
// parseExpression flattens tree expressions into stack-based post-fix operations
func (p *Parser) parseExpression(expr ast.Expr) error {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name == "true" {
			p.emit("i32.const 1") // wasm uses 1 for true
			return nil
		}

		if e.Name == "false" {
			p.emit("i32.const 0") // wasm uses 0 for false
			return nil
		}

		wasmName, ok := p.locals[e.Name]
		if !ok {
			return fmt.Errorf("unknown identifier: %s", e.Name)
		}
		p.emit("local.get %s", wasmName)
		return nil

	case *ast.BasicLit:
		if e.Kind == token.INT {
			p.emit("i32.const %s", e.Value)
			return nil
		}
		return fmt.Errorf("unsupported literal kind: %v", e.Kind)

	case *ast.BinaryExpr:
		// Push left operand to stack
		if err := p.parseExpression(e.X); err != nil {
			return err
		}
		// Push right operand to stack
		if err := p.parseExpression(e.Y); err != nil {
			return err
		}

		// Emit arithmetic operation
		switch e.Op {
		case token.ADD:
			p.emit("i32.add")
		case token.SUB:
			p.emit("i32.sub")
		case token.MUL:
			p.emit("i32.mul")
		case token.QUO:
			p.emit("i32.div_s") // Signed division
		case token.REM:
			p.emit("i32.rem_s") // Signed modulo
		default:
			return fmt.Errorf("unsupported operator: %s", e.Op)
		}
		return nil

	case *ast.ParenExpr:
		return p.parseExpression(e.X)

	case *ast.UnaryExpr:
		switch e.Op {
		case token.SUB:
			p.emit("i32.const 0")
			if err := p.parseExpression(e.X); err != nil {
				return err
			}
			p.emit("i32.sub")
			return nil
		case token.ADD:
			return p.parseExpression(e.X)
		case token.NOT:
			if err := p.parseExpression(e.X); err != nil {
				return err
			}
			p.emit("i32.eqz")
			return nil
		default:
			return fmt.Errorf("unsupported unary operator: %s", e.Op)
		}

	default:
		return fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// ------------------------------------------------------------------------------------------------
// Take a Go type and convert it to an equivalent / representative Wasm type.
// There's a lot of work left but I'm starting with numbers, it's easier.
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

	return "i32" // when in doubt, i32 is assumed for now
}
