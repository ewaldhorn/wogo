package ir

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
)

// builder coordinates lowering Go AST to an IR module.
type builder struct {
	packageConstants map[string]constantDef
	scopes           []scope
	constEnv         map[string]constantDef
	localsList       []Local
	nextLocalIdx     uint32
	usedWasmNames    map[string]bool
	returnType       string
	instructions     []Instruction
	controlStack     []controlFrameKind
}

// BuildModule parses Go source and compiles it into an intermediate representation (IR) Module.
func BuildModule(src string) (*Module, error) {
	fset := token.NewFileSet()
	node, err := goparser.ParseFile(fset, "calc.go", src, 0)
	if err != nil {
		return nil, fmt.Errorf("parse go source: %w", err)
	}

	b := &builder{
		packageConstants: make(map[string]constantDef),
	}

	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
			if err := mapConstants(genDecl, b.packageConstants); err != nil {
				return nil, err
			}
		}
	}

	for _, cDef := range b.packageConstants {
		if cDef.explicitType != "" {
			var assignErr error
			b.withConstEnv(cDef, func() {
				assignErr = b.checkAssignable(cDef.expr, cDef.explicitType)
			})
			if assignErr != nil {
				return nil, assignErr
			}
		}
	}

	mod := &Module{}
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		compiledFn, err := b.buildFunction(fn)
		if err != nil {
			return nil, fmt.Errorf("compile function %s: %w", fn.Name.Name, err)
		}
		mod.Functions = append(mod.Functions, *compiledFn)
	}

	return mod, nil
}

func (b *builder) emit(insts ...Instruction) {
	b.instructions = append(b.instructions, insts...)
}

func (b *builder) emitLocalGet(binding localBinding) {
	b.emit(Instruction{
		Op:        OpLocalGet,
		LocalIdx:  binding.idx,
		LocalName: binding.wasmName,
	})
}

func (b *builder) emitLocalSet(binding localBinding) {
	b.emit(Instruction{
		Op:        OpLocalSet,
		LocalIdx:  binding.idx,
		LocalName: binding.wasmName,
	})
}

func (b *builder) buildFunction(fn *ast.FuncDecl) (*Function, error) {
	b.scopes = nil
	b.constEnv = nil
	b.nextLocalIdx = 0
	b.usedWasmNames = make(map[string]bool)
	b.localsList = nil
	b.controlStack = nil
	b.returnType = "int32"
	b.instructions = nil

	b.pushScope()
	defer b.popScope()

	function := &Function{Name: fn.Name.Name}

	if err := b.bindParams(fn, function); err != nil {
		return nil, err
	}
	if err := b.bindResults(fn, function); err != nil {
		return nil, err
	}

	if fn.Body != nil {
		for _, stmt := range fn.Body.List {
			if err := b.buildStatement(stmt); err != nil {
				return nil, err
			}
		}
	}

	if len(function.Results) > 0 {
		if len(b.instructions) == 0 || b.instructions[len(b.instructions)-1].Op != OpReturn {
			b.emit(Instruction{Op: OpUnreachable})
		}
	}

	function.Locals = b.localsList
	function.Instructions = b.instructions
	return function, nil
}

func (b *builder) bindParams(fn *ast.FuncDecl, function *Function) error {
	if fn.Type.Params == nil {
		return nil
	}
	for _, field := range fn.Type.Params.List {
		goType, err := typeName(field.Type)
		if err != nil {
			return err
		}
		wasmType, err := ParseGoType(goType)
		if err != nil {
			return err
		}
		for _, name := range field.Names {
			idx := b.nextLocalIdx
			b.nextLocalIdx++
			wasmName := b.allocWasmName(name.Name)
			if name.Name != "_" {
				if b.currentScope().has(name.Name) {
					return fmt.Errorf("%s redeclared in this block", name.Name)
				}
				b.currentScope().vars[name.Name] = localBinding{
					idx:      idx,
					wasmName: wasmName,
					goType:   goType,
				}
			}
			function.Params = append(function.Params, Param{
				Name:   wasmName,
				GoType: goType,
				Type:   wasmType,
			})
		}
	}
	return nil
}

func (b *builder) bindResults(fn *ast.FuncDecl, function *Function) error {
	if fn.Type.Results == nil {
		return nil
	}
	for _, field := range fn.Type.Results.List {
		goType, err := typeName(field.Type)
		if err != nil {
			return err
		}
		wasmType, err := ParseGoType(goType)
		if err != nil {
			return err
		}
		b.returnType = goType
		function.ReturnType = b.returnType
		function.Results = append(function.Results, wasmType)
	}
	return nil
}
