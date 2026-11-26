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
	env          map[string]constantDef
}

// ------------------------------------------------------------------------------------------------
type localBinding struct {
	idx      uint32
	wasmName string
	goType   string
}

// ------------------------------------------------------------------------------------------------
type identKind int

const (
	identNone identKind = iota
	identVar
	identConst
)

// ------------------------------------------------------------------------------------------------
type scope struct {
	vars   map[string]localBinding
	consts map[string]constantDef
}

func (s *scope) has(name string) bool {
	if _, ok := s.vars[name]; ok {
		return true
	}
	if _, ok := s.consts[name]; ok {
		return true
	}
	return false
}

// ------------------------------------------------------------------------------------------------
type controlFrameKind int

const (
	frameIf controlFrameKind = iota
	frameLoopExit
	frameLoop
	frameLoopCont
)

// ------------------------------------------------------------------------------------------------
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

// ------------------------------------------------------------------------------------------------
func (b *builder) pushScope() {
	b.scopes = append(b.scopes, scope{
		vars:   make(map[string]localBinding),
		consts: make(map[string]constantDef),
	})
}

func (b *builder) popScope() {
	b.scopes = b.scopes[:len(b.scopes)-1]
}

func (b *builder) currentScope() *scope {
	return &b.scopes[len(b.scopes)-1]
}

func (b *builder) allocWasmName(goName string) string {
	if !b.usedWasmNames[goName] {
		b.usedWasmNames[goName] = true
		return goName
	}
	for n := 1; ; n++ {
		candidate := fmt.Sprintf("%s_%d", goName, n)
		if !b.usedWasmNames[candidate] {
			b.usedWasmNames[candidate] = true
			return candidate
		}
	}
}

func (b *builder) declareLocal(goName, goType string, wasmType Type) localBinding {
	idx := b.nextLocalIdx
	b.nextLocalIdx++
	wasmName := b.allocWasmName(goName)
	binding := localBinding{
		idx:      idx,
		wasmName: wasmName,
		goType:   goType,
	}
	b.currentScope().vars[goName] = binding
	b.localsList = append(b.localsList, Local{
		Name:   wasmName,
		GoType: goType,
		Type:   wasmType,
	})
	return binding
}

func (b *builder) allocTempLocal(prefix, goType string, wasmType Type) localBinding {
	idx := b.nextLocalIdx
	b.nextLocalIdx++
	wasmName := b.allocWasmName(prefix)
	b.localsList = append(b.localsList, Local{
		Name:   wasmName,
		GoType: goType,
		Type:   wasmType,
	})
	return localBinding{
		idx:      idx,
		wasmName: wasmName,
		goType:   goType,
	}
}

func isIntegerType(goType string) bool {
	switch goType {
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return true
	default:
		return false
	}
}

func (b *builder) checkAssignable(expr ast.Expr, targetType string) error {
	resolved := b.resolveExprType(expr, targetType)
	if resolved == "" {
		return nil
	}
	if resolved != targetType {
		return fmt.Errorf("cannot use %s as %s in assignment", resolved, targetType)
	}
	if targetType == "bool" && b.resolveExprType(expr, "") == "" {
		return fmt.Errorf("cannot use untyped int as bool in assignment")
	}
	return nil
}

func (b *builder) lookupIdent(name string) (localBinding, constantDef, identKind) {
	if b.constEnv != nil {
		if cDef, ok := b.constEnv[name]; ok {
			return localBinding{}, cDef, identConst
		}
		return localBinding{}, constantDef{}, identNone
	}
	for i := len(b.scopes) - 1; i >= 0; i-- {
		if v, ok := b.scopes[i].vars[name]; ok {
			return v, constantDef{}, identVar
		}
		if c, ok := b.scopes[i].consts[name]; ok {
			return localBinding{}, c, identConst
		}
	}
	if c, ok := b.packageConstants[name]; ok {
		return localBinding{}, c, identConst
	}
	return localBinding{}, constantDef{}, identNone
}

func (b *builder) snapshotConstants() map[string]constantDef {
	env := make(map[string]constantDef, len(b.packageConstants))
	for k, v := range b.packageConstants {
		env[k] = v
	}
	for i := 0; i < len(b.scopes); i++ {
		for k := range b.scopes[i].vars {
			delete(env, k)
		}
		for k, v := range b.scopes[i].consts {
			env[k] = v
		}
	}
	return env
}

func (b *builder) withConstEnv(cDef constantDef, fn func()) {
	prev := b.constEnv
	if cDef.env != nil {
		b.constEnv = cDef.env
	} else {
		b.constEnv = b.packageConstants
	}
	fn()
	b.constEnv = prev
}

func (b *builder) inLoop() bool {
	for _, f := range b.controlStack {
		if f == frameLoop {
			return true
		}
	}
	return false
}

func (b *builder) zeroConstInstruction(wasmType Type) Instruction {
	switch wasmType {
	case TypeI64:
		return Instruction{Op: OpI64Const, I64: 0}
	case TypeF32:
		return Instruction{Op: OpF32Const, F32: 0}
	case TypeF64:
		return Instruction{Op: OpF64Const, F64: 0}
	default:
		return Instruction{Op: OpI32Const, I32: 0}
	}
}

// ------------------------------------------------------------------------------------------------
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
				idx := b.nextLocalIdx
				b.nextLocalIdx++
				wasmName := b.allocWasmName(name.Name)
				if name.Name != "_" {
					if b.currentScope().has(name.Name) {
						return nil, fmt.Errorf("%s redeclared in this block", name.Name)
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

	if len(function.Results) > 0 {
		if len(b.instructions) == 0 || b.instructions[len(b.instructions)-1].Op != OpReturn {
			b.instructions = append(b.instructions, Instruction{Op: OpUnreachable})
		}
	}

	function.Locals = b.localsList
	function.Instructions = b.instructions
	return function, nil
}

// ------------------------------------------------------------------------------------------------
// branchDepth returns the relative WebAssembly label index for the target frame kind,
// searching from innermost (top of stack) to outermost.
func (b *builder) branchDepth(target controlFrameKind) (uint32, error) {
	for i := len(b.controlStack) - 1; i >= 0; i-- {
		if b.controlStack[i] == target {
			return uint32(len(b.controlStack) - 1 - i), nil
		}
	}
	return 0, fmt.Errorf("target control frame not found")
}

// ------------------------------------------------------------------------------------------------
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
		b.instructions = append(b.instructions, Instruction{Op: OpReturn})
		return nil
	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// ------------------------------------------------------------------------------------------------
// buildIfStmt compiles an if statement, including optional init statement and else branch.
func (b *builder) buildIfStmt(s *ast.IfStmt) error {
	b.pushScope()
	defer b.popScope()

	if s.Init != nil {
		if err := b.buildStatement(s.Init); err != nil {
			return err
		}
	}

	condType := b.resolveExprType(s.Cond, "bool")
	if condType != "bool" {
		return fmt.Errorf("non-bool %s used as if condition", condType)
	}

	if err := b.compileExpr(s.Cond, "bool"); err != nil {
		return err
	}
	if b.resolveExprType(s.Cond, "") == "" {
		return fmt.Errorf("non-bool untyped int used as if condition")
	}

	b.controlStack = append(b.controlStack, frameIf)
	b.instructions = append(b.instructions, Instruction{Op: OpIf})

	if err := b.buildStatement(s.Body); err != nil {
		return err
	}

	if s.Else != nil {
		b.instructions = append(b.instructions, Instruction{Op: OpElse})
		if err := b.buildStatement(s.Else); err != nil {
			return err
		}
	}

	b.controlStack = b.controlStack[:len(b.controlStack)-1]
	b.instructions = append(b.instructions, Instruction{Op: OpEnd})
	return nil
}

// ------------------------------------------------------------------------------------------------
// buildForStmt compiles a for loop (init; cond; post; body) into WebAssembly block/loop structure.
// Structured layout:
//
//	[init]
//	block (exit)
//	  loop (head)
//	    [cond] -> eqz -> br_if (exit)
//	    block (continue)
//	      [body]
//	    end
//	    [post]
//	    br (head)
//	  end
//	end
func (b *builder) buildForStmt(s *ast.ForStmt) error {
	b.pushScope()
	defer b.popScope()

	if s.Init != nil {
		if err := b.buildStatement(s.Init); err != nil {
			return err
		}
	}

	// Outer block for break: branching here exits the loop.
	b.controlStack = append(b.controlStack, frameLoopExit)
	b.instructions = append(b.instructions, Instruction{Op: OpBlock})

	// Loop header: branching here restarts the loop iteration.
	b.controlStack = append(b.controlStack, frameLoop)
	b.instructions = append(b.instructions, Instruction{Op: OpLoop})

	// Condition check: if condition is false, break out to loop exit.
	if s.Cond != nil {
		condType := b.resolveExprType(s.Cond, "bool")
		if condType != "bool" {
			return fmt.Errorf("non-bool %s used as for condition", condType)
		}
		if err := b.compileExpr(s.Cond, "bool"); err != nil {
			return err
		}
		if b.resolveExprType(s.Cond, "") == "" {
			return fmt.Errorf("non-bool untyped int used as for condition")
		}
		b.instructions = append(b.instructions, Instruction{Op: OpI32Eqz})
		depth, err := b.branchDepth(frameLoopExit)
		if err != nil {
			return err
		}
		b.instructions = append(b.instructions, Instruction{Op: OpBrIf, LabelIdx: depth})
	}

	// Inner block for continue: branching here jumps to the post statement.
	b.controlStack = append(b.controlStack, frameLoopCont)
	b.instructions = append(b.instructions, Instruction{Op: OpBlock})

	if err := b.buildStatement(s.Body); err != nil {
		return err
	}

	// Close inner continue block
	b.controlStack = b.controlStack[:len(b.controlStack)-1]
	b.instructions = append(b.instructions, Instruction{Op: OpEnd})

	// Execute post statement (if any)
	if s.Post != nil {
		if err := b.buildStatement(s.Post); err != nil {
			return err
		}
	}

	// Loop back to header
	loopDepth, err := b.branchDepth(frameLoop)
	if err != nil {
		return err
	}
	b.instructions = append(b.instructions, Instruction{Op: OpBr, LabelIdx: loopDepth})

	// Close loop
	b.controlStack = b.controlStack[:len(b.controlStack)-1]
	b.instructions = append(b.instructions, Instruction{Op: OpEnd})

	// Close outer block
	b.controlStack = b.controlStack[:len(b.controlStack)-1]
	b.instructions = append(b.instructions, Instruction{Op: OpEnd})

	return nil
}

// ------------------------------------------------------------------------------------------------
// buildRangeStmt compiles an integer for range loop into WebAssembly block/loop instructions.
func (b *builder) buildRangeStmt(s *ast.RangeStmt) error {
	if s.Value != nil {
		return fmt.Errorf("range over integer permits only one iteration variable")
	}

	b.pushScope()
	defer b.popScope()

	var keyIdent *ast.Ident
	var keyBinding localBinding
	hasKey := false
	defineKey := false

	if s.Key != nil {
		ident, ok := s.Key.(*ast.Ident)
		if !ok {
			return fmt.Errorf("cannot assign to non-identifier: %T", s.Key)
		}
		keyIdent = ident
		switch s.Tok {
		case token.DEFINE:
			if keyIdent.Name == "_" {
				return fmt.Errorf("no new variables on left side of :=")
			}
			hasKey = true
			defineKey = true
		case token.ASSIGN:
			if keyIdent.Name != "_" {
				binding, _, kind := b.lookupIdent(keyIdent.Name)
				switch kind {
				case identNone:
					return fmt.Errorf("unknown identifier: %s", keyIdent.Name)
				case identConst:
					return fmt.Errorf("cannot assign to constant: %s", keyIdent.Name)
				case identVar:
					keyBinding = binding
					hasKey = true
				}
			}
		default:
			return fmt.Errorf("unsupported range assignment operator: %s", s.Tok)
		}
	}

	xType := b.resolveExprType(s.X, "")
	var rangeType string
	if hasKey && !defineKey {
		if xType == "" {
			if !isIntegerType(keyBinding.goType) {
				return fmt.Errorf("cannot use range type int32 as %s in assignment", keyBinding.goType)
			}
			rangeType = keyBinding.goType
		} else {
			if !isIntegerType(xType) {
				return fmt.Errorf("cannot range over %s", xType)
			}
			if xType != keyBinding.goType {
				return fmt.Errorf("cannot use range type %s as %s in assignment", xType, keyBinding.goType)
			}
			rangeType = xType
		}
	} else {
		if xType == "" {
			xType = "int32"
		}
		if !isIntegerType(xType) {
			return fmt.Errorf("cannot range over %s", xType)
		}
		rangeType = xType
	}

	wasmType, err := ParseGoType(rangeType)
	if err != nil {
		return err
	}

	// Evaluate range limit once before entering the loop and before declaring the loop key.
	if err := b.compileExpr(s.X, rangeType); err != nil {
		return err
	}
	limitLocal := b.allocTempLocal("__range_limit", rangeType, wasmType)
	b.instructions = append(b.instructions, Instruction{
		Op:        OpLocalSet,
		LocalIdx:  limitLocal.idx,
		LocalName: limitLocal.wasmName,
	})

	idxLocal := b.allocTempLocal("__range_idx", rangeType, wasmType)
	b.instructions = append(b.instructions,
		b.zeroConstInstruction(wasmType),
		Instruction{
			Op:        OpLocalSet,
			LocalIdx:  idxLocal.idx,
			LocalName: idxLocal.wasmName,
		},
	)

	if defineKey {
		keyBinding = b.declareLocal(keyIdent.Name, rangeType, wasmType)
	}

	// Outer block for break: branching here exits the loop.
	b.controlStack = append(b.controlStack, frameLoopExit)
	b.instructions = append(b.instructions, Instruction{Op: OpBlock})

	// Loop header: branching here restarts the loop iteration.
	b.controlStack = append(b.controlStack, frameLoop)
	b.instructions = append(b.instructions, Instruction{Op: OpLoop})

	// Exit loop unless __range_idx < __range_limit.
	b.instructions = append(b.instructions,
		Instruction{Op: OpLocalGet, LocalIdx: idxLocal.idx, LocalName: idxLocal.wasmName},
		Instruction{Op: OpLocalGet, LocalIdx: limitLocal.idx, LocalName: limitLocal.wasmName},
	)
	ltOp, err := binaryOpCode(token.LSS, rangeType)
	if err != nil {
		return err
	}
	b.instructions = append(b.instructions,
		Instruction{Op: ltOp},
		Instruction{Op: OpI32Eqz},
	)
	exitDepth, err := b.branchDepth(frameLoopExit)
	if err != nil {
		return err
	}
	b.instructions = append(b.instructions, Instruction{Op: OpBrIf, LabelIdx: exitDepth})

	if hasKey {
		b.instructions = append(b.instructions,
			Instruction{Op: OpLocalGet, LocalIdx: idxLocal.idx, LocalName: idxLocal.wasmName},
			Instruction{Op: OpLocalSet, LocalIdx: keyBinding.idx, LocalName: keyBinding.wasmName},
		)
	}

	// Inner block for continue: branching here skips to incrementing __range_idx.
	b.controlStack = append(b.controlStack, frameLoopCont)
	b.instructions = append(b.instructions, Instruction{Op: OpBlock})

	if err := b.buildStatement(s.Body); err != nil {
		return err
	}

	// Close inner continue block
	b.controlStack = b.controlStack[:len(b.controlStack)-1]
	b.instructions = append(b.instructions, Instruction{Op: OpEnd})

	// Increment hidden loop counter by 1.
	b.instructions = append(b.instructions, Instruction{Op: OpLocalGet, LocalIdx: idxLocal.idx, LocalName: idxLocal.wasmName})
	if wasmType == TypeI64 {
		b.instructions = append(b.instructions, Instruction{Op: OpI64Const, I64: 1})
	} else {
		b.instructions = append(b.instructions, Instruction{Op: OpI32Const, I32: 1})
	}
	addOp, err := binaryOpCode(token.ADD, rangeType)
	if err != nil {
		return err
	}
	b.instructions = append(b.instructions,
		Instruction{Op: addOp},
		Instruction{Op: OpLocalSet, LocalIdx: idxLocal.idx, LocalName: idxLocal.wasmName},
	)

	// Loop back to header.
	loopDepth, err := b.branchDepth(frameLoop)
	if err != nil {
		return err
	}
	b.instructions = append(b.instructions, Instruction{Op: OpBr, LabelIdx: loopDepth})

	// Close loop
	b.controlStack = b.controlStack[:len(b.controlStack)-1]
	b.instructions = append(b.instructions, Instruction{Op: OpEnd})

	// Close outer block
	b.controlStack = b.controlStack[:len(b.controlStack)-1]
	b.instructions = append(b.instructions, Instruction{Op: OpEnd})

	return nil
}

// ------------------------------------------------------------------------------------------------
// buildBranchStmt compiles break and continue statements.
func (b *builder) buildBranchStmt(s *ast.BranchStmt) error {
	switch s.Tok {
	case token.BREAK:
		depth, err := b.branchDepth(frameLoopExit)
		if err != nil {
			return fmt.Errorf("break is not in a loop")
		}
		b.instructions = append(b.instructions, Instruction{Op: OpBr, LabelIdx: depth})
		return nil
	case token.CONTINUE:
		depth, err := b.branchDepth(frameLoopCont)
		if err != nil {
			return fmt.Errorf("continue is not in a loop")
		}
		b.instructions = append(b.instructions, Instruction{Op: OpBr, LabelIdx: depth})
		return nil
	default:
		return fmt.Errorf("unsupported branch statement: %v", s.Tok)
	}
}

// ------------------------------------------------------------------------------------------------
// buildIncDecStmt compiles i++ and i-- statements.
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
	targetType := binding.goType

	// Emit: local.get $ident
	b.instructions = append(b.instructions, Instruction{
		Op:        OpLocalGet,
		LocalIdx:  binding.idx,
		LocalName: binding.wasmName,
	})

	// Emit constant 1 of targetType
	switch targetType {
	case "int32", "int", "uint32", "uint":
		b.instructions = append(b.instructions, Instruction{Op: OpI32Const, I32: 1})
	case "int64", "uint64":
		b.instructions = append(b.instructions, Instruction{Op: OpI64Const, I64: 1})
	case "float32":
		b.instructions = append(b.instructions, Instruction{Op: OpF32Const, F32: 1.0})
	case "float64":
		b.instructions = append(b.instructions, Instruction{Op: OpF64Const, F64: 1.0})
	default:
		return fmt.Errorf("unsupported type %s for ++/--", targetType)
	}

	var binOp token.Token
	if s.Tok == token.INC {
		binOp = token.ADD
	} else if s.Tok == token.DEC {
		binOp = token.SUB
	} else {
		return fmt.Errorf("unsupported inc/dec operator: %v", s.Tok)
	}

	opCode, err := binaryOpCode(binOp, targetType)
	if err != nil {
		return err
	}
	b.instructions = append(b.instructions, Instruction{Op: opCode})

	// Store result: local.set $ident
	b.instructions = append(b.instructions, Instruction{
		Op:        OpLocalSet,
		LocalIdx:  binding.idx,
		LocalName: binding.wasmName,
	})
	return nil
}

// ------------------------------------------------------------------------------------------------
// buildAssignStmt compiles assignment statements (:=, =, +=, -=, *=, /=, %=).
func (b *builder) buildAssignStmt(s *ast.AssignStmt) error {
	switch s.Tok {
	case token.DEFINE: // :=
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
		// Phase 1: compile every value before any new name enters scope.
		// A variable is not in scope in its own declaration (x := x + 1 fails or reads outer x).
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
				// A sibling in the current scope is new, so Go reuses this variable as a plain assignment.
				if err := b.compileExpr(s.Rhs[i], existing.goType); err != nil {
					return err
				}
				if err := b.checkAssignable(s.Rhs[i], existing.goType); err != nil {
					return err
				}
				pending[i] = pendingDecl{goType: existing.goType}
			} else {
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
		}
		// Phase 2: register the new names, then store in reverse order so
		// each value pops into its own target (x, y := y, x swaps).
		for i, ident := range idents {
			if !pending[i].isNew {
				continue
			}
			b.declareLocal(ident.Name, pending[i].goType, pending[i].wasmType)
		}
		for i := len(idents) - 1; i >= 0; i-- {
			if pending[i].isBlank {
				b.instructions = append(b.instructions, Instruction{Op: OpDrop})
				continue
			}
			binding := cur.vars[idents[i].Name]
			b.instructions = append(b.instructions, Instruction{
				Op:        OpLocalSet,
				LocalIdx:  binding.idx,
				LocalName: binding.wasmName,
			})
		}
		return nil

	case token.ASSIGN: // =
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
		// Compile every value first so right-hand sides all observe
		// pre-assignment state, then store in reverse order so each value
		// pops into its own target (a, b = b, a swaps).
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
		// A repeated target keeps its last value, matching Go's left-to-right
		// assignment; earlier duplicates and blanks discard their value.
		stored := make(map[uint32]bool, len(targets))
		for i := len(targets) - 1; i >= 0; i-- {
			if targets[i].isBlank || stored[targets[i].binding.idx] {
				b.instructions = append(b.instructions, Instruction{Op: OpDrop})
				continue
			}
			stored[targets[i].binding.idx] = true
			b.instructions = append(b.instructions, Instruction{
				Op:        OpLocalSet,
				LocalIdx:  targets[i].binding.idx,
				LocalName: targets[i].binding.wasmName,
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
		targetType := binding.goType

		// Emit: local.get $ident
		b.instructions = append(b.instructions, Instruction{
			Op:        OpLocalGet,
			LocalIdx:  binding.idx,
			LocalName: binding.wasmName,
		})

		// Compile RHS expression
		if err := b.compileExpr(s.Rhs[0], targetType); err != nil {
			return err
		}
		if err := b.checkAssignable(s.Rhs[0], targetType); err != nil {
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
			LocalIdx:  binding.idx,
			LocalName: binding.wasmName,
		})
		return nil

	default:
		return fmt.Errorf("unsupported assignment operator: %s", s.Tok)
	}
}

// ------------------------------------------------------------------------------------------------
// buildConstDecl registers constant declarations inside the current lexical scope.
func (b *builder) buildConstDecl(genDecl *ast.GenDecl) error {
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
			if _, err := ParseGoType(t); err != nil {
				return err
			}
			explicitType = t
		}
		if len(vspec.Values) != len(vspec.Names) {
			return fmt.Errorf("constant declaration mismatch: %d names but %d values", len(vspec.Names), len(vspec.Values))
		}
		env := b.snapshotConstants()
		for i, name := range vspec.Names {
			if explicitType != "" {
				var assignErr error
				b.withConstEnv(constantDef{env: env}, func() {
					assignErr = b.checkAssignable(vspec.Values[i], explicitType)
				})
				if assignErr != nil {
					return assignErr
				}
			}
			if name.Name == "_" {
				continue
			}
			if cur.has(name.Name) {
				return fmt.Errorf("%s redeclared in this block", name.Name)
			}
			cur.consts[name.Name] = constantDef{
				explicitType: explicitType,
				expr:         vspec.Values[i],
				env:          env,
			}
		}
	}
	return nil
}

// ------------------------------------------------------------------------------------------------
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

		// Phase 2: register non-blank locals, then store/drop values or reset to zero inside loops.
		bindings := make([]localBinding, len(pending))
		for i, p := range pending {
			if p.isBlank {
				continue
			}
			bindings[i] = b.declareLocal(p.name, p.goType, p.wasmType)
		}

		if len(vspec.Values) > 0 {
			for i := len(pending) - 1; i >= 0; i-- {
				if pending[i].isBlank {
					b.instructions = append(b.instructions, Instruction{Op: OpDrop})
					continue
				}
				b.instructions = append(b.instructions, Instruction{
					Op:        OpLocalSet,
					LocalIdx:  bindings[i].idx,
					LocalName: bindings[i].wasmName,
				})
			}
		} else if b.inLoop() {
			for i, p := range pending {
				if p.isBlank {
					continue
				}
				b.instructions = append(b.instructions,
					b.zeroConstInstruction(p.wasmType),
					Instruction{
						Op:        OpLocalSet,
						LocalIdx:  bindings[i].idx,
						LocalName: bindings[i].wasmName,
					},
				)
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
	if e.Name == "_" {
		return fmt.Errorf("cannot use _ as value")
	}

	binding, cDef, kind := b.lookupIdent(e.Name)
	switch kind {
	case identConst:
		targetType := exprType
		if cDef.explicitType != "" {
			targetType = cDef.explicitType
		}
		var compileErr error
		b.withConstEnv(cDef, func() {
			compileErr = b.compileExpr(cDef.expr, targetType)
		})
		return compileErr
	case identVar:
		b.instructions = append(b.instructions, Instruction{
			Op:        OpLocalGet,
			LocalIdx:  binding.idx,
			LocalName: binding.wasmName,
		})
		return nil
	default:
		return fmt.Errorf("unknown identifier: %s", e.Name)
	}
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
// isComparison reports whether an operator is a comparison operator.
func isComparison(op token.Token) bool {
	switch op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}

// ------------------------------------------------------------------------------------------------
func (b *builder) compileBinaryExpr(e *ast.BinaryExpr, exprType string) error {
	if isComparison(e.Op) {
		operandType, err := b.resolveComparisonOperandType(e)
		if err != nil {
			return err
		}
		if err := b.compileExpr(e.X, operandType); err != nil {
			return err
		}
		if err := b.compileExpr(e.Y, operandType); err != nil {
			return err
		}
		opCode, err := binaryOpCode(e.Op, operandType)
		if err != nil {
			return err
		}
		b.instructions = append(b.instructions, Instruction{Op: opCode})
		return nil
	}

	cX := b.concreteExprType(e.X)
	cY := b.concreteExprType(e.Y)
	if cX != "" && cY != "" && cX != cY {
		return fmt.Errorf("invalid operation: %s (mismatched types %s and %s)", e.Op, cX, cY)
	}
	tX := b.resolveExprType(e.X, exprType)
	tY := b.resolveExprType(e.Y, exprType)
	if tX != "" && tY != "" && tX != tY {
		return fmt.Errorf("invalid operation: %s (mismatched types %s and %s)", e.Op, tX, tY)
	}

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
		b.instructions = append(b.instructions, Instruction{Op: OpI32Eqz})
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
		binding, cDef, kind := b.lookupIdent(e.Name)
		switch kind {
		case identVar:
			return binding.goType
		case identConst:
			if cDef.explicitType != "" {
				return cDef.explicitType
			}
			var resolved string
			b.withConstEnv(cDef, func() {
				resolved = b.resolveExprType(cDef.expr, targetType)
			})
			return resolved
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
		if isComparison(e.Op) {
			return "bool"
		}
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
// resolveComparisonOperandType determines the common operand type for a comparison.
func (b *builder) resolveComparisonOperandType(e *ast.BinaryExpr) (string, error) {
	cX := b.concreteExprType(e.X)
	cY := b.concreteExprType(e.Y)
	if cX != "" && cY != "" && cX != cY {
		return "", fmt.Errorf("invalid operation: %s (mismatched types %s and %s)", e.Op, cX, cY)
	}

	target := cX
	if target == "" {
		target = cY
	}

	tX := b.resolveExprType(e.X, target)
	tY := b.resolveExprType(e.Y, target)

	if tX != "" && tY != "" && tX != tY {
		return "", fmt.Errorf("invalid operation: %s (mismatched types %s and %s)", e.Op, tX, tY)
	}

	if tX != "" && tX != "int32" {
		return tX, nil
	}
	if tY != "" && tY != "int32" {
		return tY, nil
	}
	if tX != "" {
		return tX, nil
	}
	if tY != "" {
		return tY, nil
	}
	return "int32", nil
}

// ------------------------------------------------------------------------------------------------
// concreteExprType returns the explicit or declared type of an expression if known,
// or empty string if it is an untyped literal or unconstrained expression.
func (b *builder) concreteExprType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name == "true" || e.Name == "false" {
			return "bool"
		}
		binding, cDef, kind := b.lookupIdent(e.Name)
		switch kind {
		case identVar:
			return binding.goType
		case identConst:
			if cDef.explicitType != "" {
				return cDef.explicitType
			}
		}
	case *ast.ParenExpr:
		return b.concreteExprType(e.X)
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return "bool"
		}
		return b.concreteExprType(e.X)
	case *ast.BinaryExpr:
		if isComparison(e.Op) {
			return "bool"
		}
		if c := b.concreteExprType(e.X); c != "" {
			return c
		}
		return b.concreteExprType(e.Y)
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok && x.Name == "math" {
			switch e.Sel.Name {
			case "Pi", "E", "Phi":
				return "float64"
			}
		}
	}
	return ""
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
			if name.Name == "_" {
				continue
			}
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
	case "bool":
		switch op {
		case token.EQL:
			return OpI32Eq, nil
		case token.NEQ:
			return OpI32Ne, nil
		}
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
		case token.EQL:
			return OpI64Eq, nil
		case token.NEQ:
			return OpI64Ne, nil
		case token.LSS:
			return OpI64LtS, nil
		case token.LEQ:
			return OpI64LeS, nil
		case token.GTR:
			return OpI64GtS, nil
		case token.GEQ:
			return OpI64GeS, nil
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
		case token.EQL:
			return OpI64Eq, nil
		case token.NEQ:
			return OpI64Ne, nil
		case token.LSS:
			return OpI64LtU, nil
		case token.LEQ:
			return OpI64LeU, nil
		case token.GTR:
			return OpI64GtU, nil
		case token.GEQ:
			return OpI64GeU, nil
		}
	case "int32", "int":
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
		case token.EQL:
			return OpI32Eq, nil
		case token.NEQ:
			return OpI32Ne, nil
		case token.LSS:
			return OpI32LtS, nil
		case token.LEQ:
			return OpI32LeS, nil
		case token.GTR:
			return OpI32GtS, nil
		case token.GEQ:
			return OpI32GeS, nil
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
		case token.EQL:
			return OpI32Eq, nil
		case token.NEQ:
			return OpI32Ne, nil
		case token.LSS:
			return OpI32LtU, nil
		case token.LEQ:
			return OpI32LeU, nil
		case token.GTR:
			return OpI32GtU, nil
		case token.GEQ:
			return OpI32GeU, nil
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
		case token.EQL:
			return OpF32Eq, nil
		case token.NEQ:
			return OpF32Ne, nil
		case token.LSS:
			return OpF32Lt, nil
		case token.LEQ:
			return OpF32Le, nil
		case token.GTR:
			return OpF32Gt, nil
		case token.GEQ:
			return OpF32Ge, nil
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
		case token.EQL:
			return OpF64Eq, nil
		case token.NEQ:
			return OpF64Ne, nil
		case token.LSS:
			return OpF64Lt, nil
		case token.LEQ:
			return OpF64Le, nil
		case token.GTR:
			return OpF64Gt, nil
		case token.GEQ:
			return OpF64Ge, nil
		}
	default:
		return 0, fmt.Errorf("unsupported type %q for binary operator %s", exprType, op)
	}
	return 0, fmt.Errorf("unsupported binary operator %s for type %s", op, exprType)
}
