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
type constantDef struct {
	constType    string
	constantExpr ast.Expr
}

// ------------------------------------------------------------------------------------------------
type Parser struct {
	buf              bytes.Buffer
	indent           int
	locals           map[string]string // Map Go to Wasm local/param
	localTypes       map[string]string
	localConstants   map[string]constantDef // can shadow global constants doh!
	packageConstants map[string]constantDef
	returnType       string
}

// ------------------------------------------------------------------------------------------------
func (p *Parser) Parse(src string) string {
	fset := token.NewFileSet()
	node, err := goparser.ParseFile(fset, "calc.go", src, 0)
	if err != nil {
		panic(err)
	}

	p.packageConstants = make(map[string]constantDef)
	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
			mapConstants(genDecl, p.packageConstants)
		}
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
	p.localConstants = make(map[string]constantDef)
	if p.packageConstants == nil {
		p.packageConstants = make(map[string]constantDef)
	}
	p.returnType = "int32"
	functionName := funcDecl.Name.Name

	if funcDecl.Body != nil {
		for _, stmt := range funcDecl.Body.List {
			if declStmt, ok := stmt.(*ast.DeclStmt); ok {
				if genDecl, ok := declStmt.Decl.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
					mapConstants(genDecl, p.localConstants)
				}
			}
		}
	}

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
	case *ast.DeclStmt:
		if genDecl, ok := s.Decl.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
			return nil
		}
		return fmt.Errorf("unsupported declaration statement: %T", s.Decl)
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
// lookupConstant checks for a constant in local scope first, then package scope.
func (p *Parser) lookupConstant(name string) (constantDef, bool) {
	if c, ok := p.localConstants[name]; ok {
		return c, true
	}

	if c, ok := p.packageConstants[name]; ok {
		return c, true
	}

	return constantDef{}, false
}

// ------------------------------------------------------------------------------------------------
// mapConstants adds constant definitions from an ast.GenDecl to the specified map.
func mapConstants(genDecl *ast.GenDecl, target map[string]constantDef) {
	for _, spec := range genDecl.Specs {
		vspec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		var explicitType string
		if vspec.Type != nil {
			explicitType = typeName(vspec.Type)
		}
		for i, name := range vspec.Names {
			if i < len(vspec.Values) {
				target[name.Name] = constantDef{
					constType:    explicitType,
					constantExpr: vspec.Values[i],
				}
			}
		}
	}
}
