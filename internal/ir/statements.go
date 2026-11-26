package ir

import (
	"fmt"
	"go/ast"
	"go/token"
	"slices"
)

var compoundAssignOps = map[token.Token]token.Token{
	token.ADD_ASSIGN: token.ADD,
	token.SUB_ASSIGN: token.SUB,
	token.MUL_ASSIGN: token.MUL,
	token.QUO_ASSIGN: token.QUO,
	token.REM_ASSIGN: token.REM,
}

func (b *builder) buildStatement(stmt ast.Stmt) error {
	switch s := stmt.(type) {
	case *ast.EmptyStmt:
		return nil
	case *ast.BlockStmt:
		b.pushScope()
		defer b.popScope()
		for _, inner := range s.List {
			if err := b.buildStatement(inner); err != nil {
				return err
			}
		}
		return nil
	case *ast.IfStmt:
		return b.buildIfStmt(s)
	case *ast.ForStmt:
		return b.buildForStmt(s)
	case *ast.RangeStmt:
		return b.buildRangeStmt(s)
	case *ast.BranchStmt:
		return b.buildBranchStmt(s)
	case *ast.IncDecStmt:
		return b.buildIncDecStmt(s)
	case *ast.DeclStmt:
		genDecl, ok := s.Decl.(*ast.GenDecl)
		if !ok {
			return fmt.Errorf("unsupported declaration statement: %T", s.Decl)
		}
		if genDecl.Tok == token.CONST {
			return b.buildConstDecl(genDecl)
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
			if b.returnType != "" {
				if err := b.checkAssignable(expr, b.returnType); err != nil {
					return err
				}
			}
		}
		b.emit(Instruction{Op: OpReturn})
		return nil
	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// buildIncDecStmt compiles ++ and decrement statements.
func (b *builder) buildIncDecStmt(s *ast.IncDecStmt) error {
	ident, ok := s.X.(*ast.Ident)
	if !ok {
		return fmt.Errorf("cannot increment/decrement non-identifier: %T", s.X)
	}
	if ident.Name == "_" {
		return fmt.Errorf("cannot use _ as value")
	}
	binding, _, kind := b.lookupIdent(ident.Name)
	switch kind {
	case identNone:
		return fmt.Errorf("unknown identifier: %s", ident.Name)
	case identConst:
		return fmt.Errorf("cannot increment/decrement constant: %s", ident.Name)
	}

	b.emitLocalGet(binding)

	oneInst, err := oneConstInstruction(binding.goType)
	if err != nil {
		return err
	}
	b.emit(oneInst)

	var binOp token.Token
	switch s.Tok {
	case token.INC:
		binOp = token.ADD
	case token.DEC:
		binOp = token.SUB
	default:
		return fmt.Errorf("unsupported inc/dec operator: %v", s.Tok)
	}

	opCode, err := binaryOpCode(binOp, binding.goType)
	if err != nil {
		return err
	}
	b.emit(Instruction{Op: opCode})
	b.emitLocalSet(binding)
	return nil
}

// buildAssignStmt compiles assignment statements (:=, =, and compound forms).
func (b *builder) buildAssignStmt(s *ast.AssignStmt) error {
	switch s.Tok {
	case token.DEFINE:
		return b.buildShortVarDecl(s)
	case token.ASSIGN:
		return b.buildPlainAssign(s)
	default:
		if binOp, ok := compoundAssignOps[s.Tok]; ok {
			return b.buildCompoundAssign(s, binOp)
		}
		return fmt.Errorf("unsupported assignment operator: %s", s.Tok)
	}
}

func (b *builder) buildShortVarDecl(s *ast.AssignStmt) error {
	if len(s.Lhs) != len(s.Rhs) {
		return fmt.Errorf("assignment mismatch: %d variables but %d values", len(s.Lhs), len(s.Rhs))
	}
	cur := b.currentScope()
	idents := make([]*ast.Ident, len(s.Lhs))
	seen := make(map[string]bool, len(s.Lhs))
	hasNew := false
	for i, lhs := range s.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			return fmt.Errorf("cannot assign to non-identifier: %T", lhs)
		}
		idents[i] = ident
		if ident.Name == "_" {
			continue
		}
		if seen[ident.Name] {
			return fmt.Errorf("%s repeated on left side of :=", ident.Name)
		}
		seen[ident.Name] = true
		if _, isConst := cur.consts[ident.Name]; isConst {
			return fmt.Errorf("%s redeclared in this block", ident.Name)
		}
		if _, exists := cur.vars[ident.Name]; !exists {
			hasNew = true
		}
	}
	if !hasNew {
		return fmt.Errorf("no new variables on left side of :=")
	}

	// Phase 1: compile every value before any new name enters scope so x := x + 1 reads outer x or fails.
	type pendingDecl struct {
		goType   string
		wasmType Type
		isNew    bool
		isBlank  bool
	}
	pending := make([]pendingDecl, len(idents))
	for i, ident := range idents {
		if ident.Name == "_" {
			rhsType := b.resolveExprType(s.Rhs[i], "")
			if rhsType == "" {
				rhsType = "int32"
			}
			if err := b.compileExpr(s.Rhs[i], rhsType); err != nil {
				return err
			}
			pending[i] = pendingDecl{goType: rhsType, isBlank: true}
			continue
		}
		if existing, exists := cur.vars[ident.Name]; exists {
			if err := b.compileExpr(s.Rhs[i], existing.goType); err != nil {
				return err
			}
			if err := b.checkAssignable(s.Rhs[i], existing.goType); err != nil {
				return err
			}
			pending[i] = pendingDecl{goType: existing.goType}
			continue
		}
		rhsType := b.resolveExprType(s.Rhs[i], "")
		if rhsType == "" {
			rhsType = "int32"
		}
		wasmType, err := ParseGoType(rhsType)
		if err != nil {
			return err
		}
		if err := b.compileExpr(s.Rhs[i], rhsType); err != nil {
			return err
		}
		pending[i] = pendingDecl{goType: rhsType, wasmType: wasmType, isNew: true}
	}

	// Phase 2: register the new names, then store in reverse order so each pops into its own target.
	for i, ident := range idents {
		if pending[i].isNew {
			b.declareLocal(ident.Name, pending[i].goType, pending[i].wasmType)
		}
	}
	for i, ident := range slices.Backward(idents) {
		if pending[i].isBlank {
			b.emit(Instruction{Op: OpDrop})
			continue
		}
		b.emitLocalSet(cur.vars[ident.Name])
	}
	return nil
}

func (b *builder) buildPlainAssign(s *ast.AssignStmt) error {
	if len(s.Lhs) != len(s.Rhs) {
		return fmt.Errorf("assignment mismatch: %d variables but %d values", len(s.Lhs), len(s.Rhs))
	}
	type assignTarget struct {
		isBlank bool
		binding localBinding
	}
	targets := make([]assignTarget, len(s.Lhs))
	for i, lhs := range s.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			return fmt.Errorf("cannot assign to non-identifier: %T", lhs)
		}
		if ident.Name == "_" {
			targets[i] = assignTarget{isBlank: true}
			continue
		}
		binding, _, kind := b.lookupIdent(ident.Name)
		switch kind {
		case identNone:
			return fmt.Errorf("unknown identifier: %s", ident.Name)
		case identConst:
			return fmt.Errorf("cannot assign to constant: %s", ident.Name)
		}
		targets[i] = assignTarget{binding: binding}
	}

	// Evaluate all values before storing so aliased swaps (a, b = b, a) observe pre assignment state.
	for i, target := range targets {
		if target.isBlank {
			rhsType := b.resolveExprType(s.Rhs[i], "")
			if rhsType == "" {
				rhsType = "int32"
			}
			if err := b.compileExpr(s.Rhs[i], rhsType); err != nil {
				return err
			}
			continue
		}
		if err := b.compileExpr(s.Rhs[i], target.binding.goType); err != nil {
			return err
		}
		if err := b.checkAssignable(s.Rhs[i], target.binding.goType); err != nil {
			return err
		}
	}

	stored := make(map[uint32]bool, len(targets))
	for _, target := range slices.Backward(targets) {
		if target.isBlank || stored[target.binding.idx] {
			b.emit(Instruction{Op: OpDrop})
			continue
		}
		stored[target.binding.idx] = true
		b.emitLocalSet(target.binding)
	}
	return nil
}

func (b *builder) buildCompoundAssign(s *ast.AssignStmt, binOp token.Token) error {
	if len(s.Lhs) != 1 || len(s.Rhs) != 1 {
		return fmt.Errorf("compound assignment supports single variable only")
	}
	ident, ok := s.Lhs[0].(*ast.Ident)
	if !ok {
		return fmt.Errorf("cannot assign to non-identifier: %T", s.Lhs[0])
	}
	if ident.Name == "_" {
		return fmt.Errorf("cannot use _ as value")
	}
	binding, _, kind := b.lookupIdent(ident.Name)
	switch kind {
	case identNone:
		return fmt.Errorf("unknown identifier: %s", ident.Name)
	case identConst:
		return fmt.Errorf("cannot assign to constant: %s", ident.Name)
	}

	b.emitLocalGet(binding)
	if err := b.compileExpr(s.Rhs[0], binding.goType); err != nil {
		return err
	}
	if err := b.checkAssignable(s.Rhs[0], binding.goType); err != nil {
		return err
	}

	opCode, err := binaryOpCode(binOp, binding.goType)
	if err != nil {
		return err
	}
	b.emit(Instruction{Op: opCode})
	b.emitLocalSet(binding)
	return nil
}

// buildVarDecl compiles var declaration statements inside function bodies.
func (b *builder) buildVarDecl(genDecl *ast.GenDecl) error {
	cur := b.currentScope()
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

		seen := make(map[string]bool, len(vspec.Names))
		type pendingVar struct {
			name     string
			goType   string
			wasmType Type
			isBlank  bool
		}
		pending := make([]pendingVar, len(vspec.Names))

		// Phase 1: validate names and compile every value before new names enter scope.
		for i, name := range vspec.Names {
			isBlank := name.Name == "_"
			if !isBlank {
				if seen[name.Name] || cur.has(name.Name) {
					return fmt.Errorf("%s redeclared in this block", name.Name)
				}
				seen[name.Name] = true
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

			if i < len(vspec.Values) {
				if err := b.compileExpr(vspec.Values[i], goType); err != nil {
					return err
				}
				if explicitType != "" {
					if err := b.checkAssignable(vspec.Values[i], explicitType); err != nil {
						return err
					}
				}
			}
			pending[i] = pendingVar{name: name.Name, goType: goType, wasmType: wasmType, isBlank: isBlank}
		}

		// Phase 2: register non blank locals, then store/drop values or reset to zero inside loops.
		bindings := make([]localBinding, len(pending))
		for i, p := range pending {
			if !p.isBlank {
				bindings[i] = b.declareLocal(p.name, p.goType, p.wasmType)
			}
		}

		if len(vspec.Values) > 0 {
			for i, p := range slices.Backward(pending) {
				if p.isBlank {
					b.emit(Instruction{Op: OpDrop})
					continue
				}
				b.emitLocalSet(bindings[i])
			}
		} else if b.inLoop() {
			for i, p := range pending {
				if p.isBlank {
					continue
				}
				b.emit(b.zeroConstInstruction(p.wasmType))
				b.emitLocalSet(bindings[i])
			}
		}
	}
	return nil
}
