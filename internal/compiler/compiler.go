package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ------------------------------------------------------------------------------------------------
type Compiler struct {
	buf    bytes.Buffer
	indent int
	locals map[string]string // Map Go to Wasm local/param
}

// ------------------------------------------------------------------------------------------------
func (c *Compiler) Compile(src string) string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "calc.go", src, 0)
	if err != nil {
		panic(err)
	}

	c.emit("(module")
	c.indent++

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if err := c.CompileFunc(fn); err != nil {
				panic(err)
			}
		}
	}

	c.indent--
	c.emit(")")

	return c.buf.String()
}

// ------------------------------------------------------------------------------------------------
// Helper to indent to the current indention level
func (c *Compiler) emit(format string, args ...any) {
	c.buf.WriteString(strings.Repeat("  ", c.indent))
	fmt.Fprintf(&c.buf, format, args...)
	c.buf.WriteByte('\n')
}

// ------------------------------------------------------------------------------------------------
// Compile ast.FuncDecl into WAT syntax
func (c *Compiler) CompileFunc(funcDecl *ast.FuncDecl) error {
	c.locals = make(map[string]string)
	functionName := funcDecl.Name.Name

	var params []string
	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			wasmType := toWasmType(field.Type)
			for _, name := range field.Names {
				c.locals[name.Name] = "$" + name.Name
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

	c.emit("(func $%s (export %q) %s %s", functionName, functionName, paramStr, resultStr)
	c.indent++

	for _, stmt := range funcDecl.Body.List {
		if err := c.compileStatement(stmt); err != nil {
			return err
		}
	}

	c.indent--
	c.emit(")")
	return nil
}

// ------------------------------------------------------------------------------------------------
func (c *Compiler) compileStatement(stmt ast.Stmt) error {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		for _, expr := range s.Results {
			if err := c.compileExpression(expr); err != nil {
				return err
			}
		}
		c.emit("return")
		return nil
	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// ------------------------------------------------------------------------------------------------
// compileExpression flattens tree expressions into stack-based post-fix operations
func (c *Compiler) compileExpression(expr ast.Expr) error {
	switch e := expr.(type) {
	case *ast.Ident:
		wasmName, ok := c.locals[e.Name]
		if !ok {
			return fmt.Errorf("unknown identifier: %s", e.Name)
		}
		c.emit("local.get %s", wasmName)
		return nil

	case *ast.BasicLit:
		if e.Kind == token.INT {
			c.emit("i32.const %s", e.Value)
			return nil
		}
		return fmt.Errorf("unsupported literal kind: %v", e.Kind)

	case *ast.BinaryExpr:
		// Push left operand to stack
		if err := c.compileExpression(e.X); err != nil {
			return err
		}
		// Push right operand to stack
		if err := c.compileExpression(e.Y); err != nil {
			return err
		}

		// Emit arithmetic operation
		switch e.Op {
		case token.ADD:
			c.emit("i32.add")
		case token.SUB:
			c.emit("i32.sub")
		case token.MUL:
			c.emit("i32.mul")
		case token.QUO:
			c.emit("i32.div_s") // Signed division
		case token.REM:
			c.emit("i32.rem_s") // Signed modulo
		default:
			return fmt.Errorf("unsupported operator: %s", e.Op)
		}
		return nil

	case *ast.ParenExpr:
		return c.compileExpression(e.X)

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
