package ir

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strconv"
)

// ------------------------------------------------------------------------------------------------
type constantDef struct {
	explicitType string
	expr         ast.Expr
}

// ------------------------------------------------------------------------------------------------
// builder coordinates lowering Go AST to an IR module.
type builder struct {
	packageConstants map[string]constantDef
	localConstants   map[string]constantDef
	locals           map[string]uint32
	localTypes       map[string]string
	localsList       []Local
	returnType       string
	instructions     []Instruction
}

// ------------------------------------------------------------------------------------------------
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

	// Find package-level constants
	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
			if err := mapConstants(genDecl, b.packageConstants); err != nil {
				return nil, err
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

// ------------------------------------------------------------------------------------------------
func (b *builder) buildFunction(fn *ast.FuncDecl) (*Function, error) {
	b.locals = make(map[string]uint32)
	b.localTypes = make(map[string]string)
	b.localConstants = make(map[string]constantDef)
	b.localsList = nil
	b.returnType = "int32"
	b.instructions = nil

	// Discover function-local constants
	if fn.Body != nil {
		for _, stmt := range fn.Body.List {
			if declStmt, ok := stmt.(*ast.DeclStmt); ok {
				if genDecl, ok := declStmt.Decl.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
					if err := mapConstants(genDecl, b.localConstants); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	function := &Function{
		Name: fn.Name.Name,
	}

	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			goType, err := typeName(field.Type)
			if err != nil {
				return nil, err
			}
			wasmType, err := ParseGoType(goType)
			if err != nil {
				return nil, err
			}
			for _, name := range field.Names {
				b.locals[name.Name] = uint32(len(b.locals))
				b.localTypes[name.Name] = goType
				function.Params = append(function.Params, Param{
					Name:   name.Name,
					GoType: goType,
					Type:   wasmType,
				})
			}
		}
	}

	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			goType, err := typeName(field.Type)
			if err != nil {
				return nil, err
			}
			wasmType, err := ParseGoType(goType)
			if err != nil {
				return nil, err
			}
			b.returnType = goType
			function.ReturnType = b.returnType
			function.Results = append(function.Results, wasmType)
		}
	}

	if fn.Body != nil {
		for _, stmt := range fn.Body.List {
			if err := b.buildStatement(stmt); err != nil {
				return nil, err
			}
		}
	}

	function.Locals = b.localsList
	function.Instructions = b.instructions
	return function, nil
}

// ------------------------------------------------------------------------------------------------
func (b *builder) buildStatement(stmt ast.Stmt) error {
	switch s := stmt.(type) {
	case *ast.DeclStmt:
		genDecl, ok := s.Decl.(*ast.GenDecl)
		if !ok {
			return fmt.Errorf("unsupported declaration statement: %T", s.Decl)
		}
		if genDecl.Tok == token.CONST {
			return nil
		}
		if genDecl.Tok == token.VAR {
			return b.buildVarDecl(genDecl)
		}
		return fmt.Errorf("unsupported declaration statement: %T", s.Decl)
	case *ast.AssignStmt:
		return b.buildAssignStmt(s)
	case *ast.ReturnStmt:
		for _, expr := range s.Results {
			if err := b.compileExpr(expr, b.returnType); err != nil {
				return err
			}
		}
		b.instructions = append(b.instructions, Instruction{Op: OpReturn})
		return nil
	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// ------------------------------------------------------------------------------------------------
// buildAssignStmt compiles assignment statements (:=, =, +=, -=, *=, /=, %=).
func (b *builder) buildAssignStmt(s *ast.AssignStmt) error {
	switch s.Tok {
	case token.DEFINE: // :=
		if len(s.Lhs) != len(s.Rhs) {
			return fmt.Errorf("assignment mismatch: %d variables but %d values", len(s.Lhs), len(s.Rhs))
		}
		idents := make([]*ast.Ident, len(s.Lhs))
		seen := make(map[string]bool, len(s.Lhs))
		hasNew := false
		for i, lhs := range s.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok {
				return fmt.Errorf("cannot assign to non-identifier: %T", lhs)
			}
			if seen[ident.Name] {
				return fmt.Errorf("%s repeated on left side of :=", ident.Name)
			}
			seen[ident.Name] = true
			idents[i] = ident
			if _, exists := b.locals[ident.Name]; !exists {
				hasNew = true
			}
		}
		if !hasNew {
			return fmt.Errorf("no new variables on left side of :=")
		}
		// Phase 1: compile every value before any new name enters scope.
		// A variable is not in scope in its own declaration (x := x + 1 fails).
		type pendingDecl struct {
			goType   string
			wasmType Type
			isNew    bool
		}
		pending := make([]pendingDecl, len(idents))
		for i, ident := range idents {
			if targetType, exists := b.localTypes[ident.Name]; exists {
				// A sibling is new, so Go reuses this variable as a plain assignment.
				pending[i] = pendingDecl{goType: targetType}
			} else {
				rhsType := b.resolveExprType(s.Rhs[i], "")
				if rhsType == "" {
					rhsType = "int32"
				}
				wasmType, err := ParseGoType(rhsType)
				if err != nil {
					return err
				}
				pending[i] = pendingDecl{goType: rhsType, wasmType: wasmType, isNew: true}
			}
			if err := b.compileExpr(s.Rhs[i], pending[i].goType); err != nil {
				return err
			}
		}
		// Phase 2: register the new names, then store in reverse order so
		// each value pops into its own target (x, y := y, x swaps).
		for i, ident := range idents {
			if !pending[i].isNew {
				continue
			}
			b.locals[ident.Name] = uint32(len(b.locals))
			b.localTypes[ident.Name] = pending[i].goType
			b.localsList = append(b.localsList, Local{
				Name:   ident.Name,
				GoType: pending[i].goType,
				Type:   pending[i].wasmType,
			})
		}
		for i := len(idents) - 1; i >= 0; i-- {
			b.instructions = append(b.instructions, Instruction{
				Op:        OpLocalSet,
				LocalIdx:  b.locals[idents[i].Name],
				LocalName: idents[i].Name,
			})
		}
		return nil

	case token.ASSIGN: // =
		if len(s.Lhs) != len(s.Rhs) {
			return fmt.Errorf("assignment mismatch: %d variables but %d values", len(s.Lhs), len(s.Rhs))
		}
		idents := make([]*ast.Ident, len(s.Lhs))
		for i, lhs := range s.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok {
				return fmt.Errorf("cannot assign to non-identifier: %T", lhs)
			}
			if _, exists := b.locals[ident.Name]; !exists {
				return fmt.Errorf("unknown identifier: %s", ident.Name)
			}
			idents[i] = ident
		}
		// Compile every value first so right-hand sides all observe
		// pre-assignment state, then store in reverse order so each value
		// pops into its own target (a, b = b, a swaps).
		for i, ident := range idents {
			if err := b.compileExpr(s.Rhs[i], b.localTypes[ident.Name]); err != nil {
				return err
			}
		}
		// A repeated target keeps its last value, matching Go's left-to-right
		// assignment; earlier duplicates only discard their value.
		stored := make(map[string]bool, len(idents))
		for i := len(idents) - 1; i >= 0; i-- {
			if stored[idents[i].Name] {
				b.instructions = append(b.instructions, Instruction{Op: OpDrop})
				continue
			}
			stored[idents[i].Name] = true
			b.instructions = append(b.instructions, Instruction{
				Op:        OpLocalSet,
				LocalIdx:  b.locals[idents[i].Name],
				LocalName: idents[i].Name,
			})
		}
		return nil

	case token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN, token.REM_ASSIGN:
		if len(s.Lhs) != 1 || len(s.Rhs) != 1 {
			return fmt.Errorf("compound assignment supports single variable only")
		}
		ident, ok := s.Lhs[0].(*ast.Ident)
		if !ok {
			return fmt.Errorf("cannot assign to non-identifier: %T", s.Lhs[0])
		}
		idx, ok := b.locals[ident.Name]
		if !ok {
			return fmt.Errorf("unknown identifier: %s", ident.Name)
		}
		targetType := b.localTypes[ident.Name]

		// Emit: local.get $ident
		b.instructions = append(b.instructions, Instruction{
			Op:        OpLocalGet,
			LocalIdx:  idx,
			LocalName: ident.Name,
		})

		// Compile RHS expression
		if err := b.compileExpr(s.Rhs[0], targetType); err != nil {
			return err
		}

		var binOp token.Token
		switch s.Tok {
		case token.ADD_ASSIGN:
			binOp = token.ADD
		case token.SUB_ASSIGN:
			binOp = token.SUB
		case token.MUL_ASSIGN:
			binOp = token.MUL
		case token.QUO_ASSIGN:
			binOp = token.QUO
		case token.REM_ASSIGN:
			binOp = token.REM
		}

		opCode, err := binaryOpCode(binOp, targetType)
		if err != nil {
			return err
		}
		b.instructions = append(b.instructions, Instruction{Op: opCode})

		// Emit: local.set $ident
		b.instructions = append(b.instructions, Instruction{
			Op:        OpLocalSet,
			LocalIdx:  idx,
			LocalName: ident.Name,
		})
		return nil

	default:
		return fmt.Errorf("unsupported assignment operator: %s", s.Tok)
	}
}

// ------------------------------------------------------------------------------------------------
// buildVarDecl compiles var declaration statements inside function bodies.
func (b *builder) buildVarDecl(genDecl *ast.GenDecl) error {
	for _, spec := range genDecl.Specs {
		vspec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		var explicitType string
		if vspec.Type != nil {
			t, err := typeName(vspec.Type)
			if err != nil {
				return err
			}
			explicitType = t
		}
		if len(vspec.Values) > 0 && len(vspec.Values) != len(vspec.Names) {
			return fmt.Errorf("assignment mismatch: %d variables but %d values", len(vspec.Names), len(vspec.Values))
		}

		for i, name := range vspec.Names {
			if _, exists := b.locals[name.Name]; exists {
				return fmt.Errorf("%s redeclared in this block", name.Name)
			}

			goType := explicitType
			if goType == "" && i < len(vspec.Values) {
				goType = b.resolveExprType(vspec.Values[i], "")
			}
			if goType == "" {
				goType = "int32"
			}
			wasmType, err := ParseGoType(goType)
			if err != nil {
				return err
			}

			// Compile the value before the name enters scope.
			if i < len(vspec.Values) {
				if err := b.compileExpr(vspec.Values[i], goType); err != nil {
					return err
				}
			}

			idx := uint32(len(b.locals))
			b.locals[name.Name] = idx
			b.localTypes[name.Name] = goType
			b.localsList = append(b.localsList, Local{
				Name:   name.Name,
				GoType: goType,
				Type:   wasmType,
			})

			if i < len(vspec.Values) {
				b.instructions = append(b.instructions, Instruction{
					Op:        OpLocalSet,
					LocalIdx:  idx,
					LocalName: name.Name,
				})
			}
		}
	}
	return nil
}

// ------------------------------------------------------------------------------------------------
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

// ------------------------------------------------------------------------------------------------
func (b *builder) compileIdent(e *ast.Ident, exprType string) error {
	if e.Name == "true" {
		b.instructions = append(b.instructions, Instruction{Op: OpI32Const, I32: 1})
		return nil
	}
	if e.Name == "false" {
		b.instructions = append(b.instructions, Instruction{Op: OpI32Const, I32: 0})
		return nil
	}

	if cDef, ok := b.lookupConstant(e.Name); ok {
		targetType := exprType
		if cDef.explicitType != "" {
			targetType = cDef.explicitType
		}
		return b.compileExpr(cDef.expr, targetType)
	}

	idx, ok := b.locals[e.Name]
	if !ok {
		return fmt.Errorf("unknown identifier: %s", e.Name)
	}
	b.instructions = append(b.instructions, Instruction{
		Op:        OpLocalGet,
		LocalIdx:  idx,
		LocalName: e.Name,
	})
	return nil
}

// ------------------------------------------------------------------------------------------------
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
			b.instructions = append(b.instructions, Instruction{
				Op:  OpI64Const,
				I64: val,
				Raw: e.Value,
			})
		case "float32":
			b.instructions = append(b.instructions, Instruction{
				Op:  OpF32Const,
				F32: float32(val),
				Raw: e.Value,
			})
		case "float64":
			b.instructions = append(b.instructions, Instruction{
				Op:  OpF64Const,
				F64: float64(val),
				Raw: e.Value,
			})
		case "int32", "int", "uint32", "uint", "bool":
			b.instructions = append(b.instructions, Instruction{
				Op:  OpI32Const,
				I32: int32(val),
				Raw: e.Value,
			})
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
			b.instructions = append(b.instructions, Instruction{
				Op:  OpF32Const,
				F32: float32(val),
				Raw: e.Value,
			})
		} else {
			b.instructions = append(b.instructions, Instruction{
				Op:  OpF64Const,
				F64: val,
				Raw: e.Value,
			})
		}
		return nil

	default:
		return fmt.Errorf("unsupported literal kind: %v", e.Kind)
	}
}

// ------------------------------------------------------------------------------------------------
func (b *builder) compileBinaryExpr(e *ast.BinaryExpr, exprType string) error {
	if err := b.compileExpr(e.X, exprType); err != nil {
		return err
	}
	if err := b.compileExpr(e.Y, exprType); err != nil {
		return err
	}

	opCode, err := binaryOpCode(e.Op, exprType)
	if err != nil {
		return err
	}
	b.instructions = append(b.instructions, Instruction{Op: opCode})
	return nil
}

// ------------------------------------------------------------------------------------------------
func (b *builder) compileUnaryExpr(e *ast.UnaryExpr, exprType string) error {
	switch e.Op {
	case token.NOT:
		if err := b.compileExpr(e.X, "bool"); err != nil {
			return err
		}
		if exprType == "int64" || exprType == "uint64" {
			b.instructions = append(b.instructions, Instruction{Op: OpI64Eqz})
		} else {
			b.instructions = append(b.instructions, Instruction{Op: OpI32Eqz})
		}
		return nil

	case token.SUB:
		if exprType == "float32" {
			if err := b.compileExpr(e.X, exprType); err != nil {
				return err
			}
			b.instructions = append(b.instructions, Instruction{Op: OpF32Neg})
			return nil
		}
		if exprType == "float64" {
			if err := b.compileExpr(e.X, exprType); err != nil {
				return err
			}
			b.instructions = append(b.instructions, Instruction{Op: OpF64Neg})
			return nil
		}

		if exprType == "int64" || exprType == "uint64" {
			b.instructions = append(b.instructions, Instruction{Op: OpI64Const, I64: 0})
			if err := b.compileExpr(e.X, exprType); err != nil {
				return err
			}
			b.instructions = append(b.instructions, Instruction{Op: OpI64Sub})
		} else {
			b.instructions = append(b.instructions, Instruction{Op: OpI32Const, I32: 0})
			if err := b.compileExpr(e.X, exprType); err != nil {
				return err
			}
			b.instructions = append(b.instructions, Instruction{Op: OpI32Sub})
		}
		return nil

	default:
		return fmt.Errorf("unsupported unary operator: %s", e.Op)
	}
}

// ------------------------------------------------------------------------------------------------
// Since we don't have a stblib yet, fake it for now.
// TODO: Introduce an actual stdlib somehow.
func (b *builder) compileSelectorExpr(e *ast.SelectorExpr, exprType string) error {
	if x, ok := e.X.(*ast.Ident); ok && x.Name == "math" {
		switch e.Sel.Name {
		case "Pi":
			b.instructions = append(b.instructions, Instruction{
				Op:  OpF64Const,
				F64: 3.141592653589793,
				Raw: "3.141592653589793",
			})
			return nil
		case "E":
			b.instructions = append(b.instructions, Instruction{
				Op:  OpF64Const,
				F64: 2.718281828459045,
				Raw: "2.718281828459045",
			})
			return nil
		case "Phi":
			b.instructions = append(b.instructions, Instruction{
				Op:  OpF64Const,
				F64: 1.618033988749895,
				Raw: "1.618033988749895",
			})
			return nil
		}
	}
	return fmt.Errorf("unsupported selector expression: %v.%s", e.X, e.Sel.Name)
}

// ------------------------------------------------------------------------------------------------
func (b *builder) resolveExprType(expr ast.Expr, targetType string) string {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name == "true" || e.Name == "false" {
			return "bool"
		}
		if t, ok := b.localTypes[e.Name]; ok {
			return t
		}
		if cDef, ok := b.lookupConstant(e.Name); ok {
			if cDef.explicitType != "" {
				return cDef.explicitType
			}
			return b.resolveExprType(cDef.expr, targetType)
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
		tX := b.resolveExprType(e.X, targetType)
		if tX != "int32" && tX != "" {
			return tX
		}
		tY := b.resolveExprType(e.Y, targetType)
		if tY != "int32" && tY != "" {
			return tY
		}
		if tX != "" {
			return tX
		}
		return targetType

	case *ast.ParenExpr:
		return b.resolveExprType(e.X, targetType)

	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return "bool"
		}
		return b.resolveExprType(e.X, targetType)

	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok && x.Name == "math" {
			switch e.Sel.Name {
			case "Pi", "E", "Phi":
				return "float64"
			}
		}
		return targetType

	default:
		return targetType
	}
}

// ------------------------------------------------------------------------------------------------
func (b *builder) lookupConstant(name string) (constantDef, bool) {
	if cDef, ok := b.localConstants[name]; ok {
		return cDef, true
	}
	if cDef, ok := b.packageConstants[name]; ok {
		return cDef, true
	}
	return constantDef{}, false
}

// ------------------------------------------------------------------------------------------------
func mapConstants(genDecl *ast.GenDecl, target map[string]constantDef) error {
	for _, spec := range genDecl.Specs {
		vspec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		var explicitType string
		if vspec.Type != nil {
			t, err := typeName(vspec.Type)
			if err != nil {
				return err
			}
			if _, err := ParseGoType(t); err != nil {
				return err
			}
			explicitType = t
		}
		for i, name := range vspec.Names {
			if i < len(vspec.Values) {
				target[name.Name] = constantDef{
					explicitType: explicitType,
					expr:         vspec.Values[i],
				}
			}
		}
	}
	return nil
}

// ------------------------------------------------------------------------------------------------
// typeName renders a Go AST type expression as a type name. Only plain
// identifiers are supported; anything else (pointers, arrays, ...) is an
// error rather than a silent int32.
func typeName(expr ast.Expr) (string, error) {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name, nil
	}
	return "", fmt.Errorf("unsupported type expression %T", expr)
}

// ------------------------------------------------------------------------------------------------
func binaryOpCode(op token.Token, exprType string) (OpCode, error) {
	switch exprType {
	case "int64":
		switch op {
		case token.ADD:
			return OpI64Add, nil
		case token.SUB:
			return OpI64Sub, nil
		case token.MUL:
			return OpI64Mul, nil
		case token.QUO:
			return OpI64DivS, nil
		case token.REM:
			return OpI64RemS, nil
		}
	case "uint64":
		switch op {
		case token.ADD:
			return OpI64Add, nil
		case token.SUB:
			return OpI64Sub, nil
		case token.MUL:
			return OpI64Mul, nil
		case token.QUO:
			return OpI64DivU, nil
		case token.REM:
			return OpI64RemU, nil
		}
	case "int32", "int", "bool":
		switch op {
		case token.ADD:
			return OpI32Add, nil
		case token.SUB:
			return OpI32Sub, nil
		case token.MUL:
			return OpI32Mul, nil
		case token.QUO:
			return OpI32DivS, nil
		case token.REM:
			return OpI32RemS, nil
		}
	case "uint32", "uint":
		switch op {
		case token.ADD:
			return OpI32Add, nil
		case token.SUB:
			return OpI32Sub, nil
		case token.MUL:
			return OpI32Mul, nil
		case token.QUO:
			return OpI32DivU, nil
		case token.REM:
			return OpI32RemU, nil
		}
	case "float32":
		switch op {
		case token.ADD:
			return OpF32Add, nil
		case token.SUB:
			return OpF32Sub, nil
		case token.MUL:
			return OpF32Mul, nil
		case token.QUO:
			return OpF32Div, nil
		}
	case "float64":
		switch op {
		case token.ADD:
			return OpF64Add, nil
		case token.SUB:
			return OpF64Sub, nil
		case token.MUL:
			return OpF64Mul, nil
		case token.QUO:
			return OpF64Div, nil
		}
	default:
		return 0, fmt.Errorf("unsupported type %q for binary operator %s", exprType, op)
	}
	return 0, fmt.Errorf("unsupported binary operator %s for type %s", op, exprType)
}
