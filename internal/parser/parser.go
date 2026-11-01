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
	buf        bytes.Buffer
	indent     int
	locals     map[string]string // Map Go to Wasm local/param
	localTypes map[string]string
	returnType string
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
	p.localTypes = make(map[string]string)
	p.returnType = "int32"
	functionName := funcDecl.Name.Name

	var params []string
	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			goType := typeName(field.Type)
			wasmType := toWasmType(field.Type)
			for _, name := range field.Names {
				p.locals[name.Name] = "$" + name.Name
				p.localTypes[name.Name] = goType
				params = append(params, fmt.Sprintf("(param $%s %s)", name.Name, wasmType))
			}
		}
	}

	var results []string
	if funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			p.returnType = typeName(field.Type)
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
