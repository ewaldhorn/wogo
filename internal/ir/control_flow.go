package ir

import (
	"fmt"
	"go/ast"
	"go/token"
	"slices"
)

type controlFrameKind int

const (
	frameIf controlFrameKind = iota
	frameLoopExit
	frameLoop
	frameLoopCont
)

type rangeKeyInfo struct {
	ident     *ast.Ident
	binding   localBinding
	hasKey    bool
	defineKey bool
}

func (b *builder) pushControl(frame controlFrameKind, op OpCode) {
	b.controlStack = append(b.controlStack, frame)
	b.emit(Instruction{Op: op})
}

func (b *builder) popControl() {
	b.controlStack = b.controlStack[:len(b.controlStack)-1]
	b.emit(Instruction{Op: OpEnd})
}

// branchDepth returns the relative WebAssembly label index for the target frame kind,
// searching from innermost (top of stack) to outermost.
func (b *builder) branchDepth(target controlFrameKind) (uint32, error) {
	for i, frame := range slices.Backward(b.controlStack) {
		if frame == target {
			return uint32(len(b.controlStack) - 1 - i), nil
		}
	}
	return 0, fmt.Errorf("target control frame not found")
}

func (b *builder) inLoop() bool {
	return slices.Contains(b.controlStack, frameLoop)
}

func (b *builder) compileBoolCondition(cond ast.Expr, stmtKind string) error {
	condType := b.resolveExprType(cond, "bool")
	if condType != "bool" {
		return fmt.Errorf("non-bool %s used as %s condition", condType, stmtKind)
	}
	if err := b.compileExpr(cond, "bool"); err != nil {
		return err
	}
	if b.resolveExprType(cond, "") == "" {
		return fmt.Errorf("non-bool untyped int used as %s condition", stmtKind)
	}
	return nil
}

// buildIfStmt compiles an if statement, including optional init statement and else branch.
func (b *builder) buildIfStmt(s *ast.IfStmt) error {
	b.pushScope()
	defer b.popScope()

	if s.Init != nil {
		if err := b.buildStatement(s.Init); err != nil {
			return err
		}
	}

	if err := b.compileBoolCondition(s.Cond, "if"); err != nil {
		return err
	}

	b.pushControl(frameIf, OpIf)

	if err := b.buildStatement(s.Body); err != nil {
		return err
	}

	if s.Else != nil {
		b.emit(Instruction{Op: OpElse})
		if err := b.buildStatement(s.Else); err != nil {
			return err
		}
	}

	b.popControl()
	return nil
}

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

	b.pushControl(frameLoopExit, OpBlock)
	b.pushControl(frameLoop, OpLoop)

	if s.Cond != nil {
		if err := b.compileBoolCondition(s.Cond, "for"); err != nil {
			return err
		}
		exitDepth, err := b.branchDepth(frameLoopExit)
		if err != nil {
			return err
		}
		b.emit(Instruction{Op: OpI32Eqz}, Instruction{Op: OpBrIf, LabelIdx: exitDepth})
	}

	b.pushControl(frameLoopCont, OpBlock)
	if err := b.buildStatement(s.Body); err != nil {
		return err
	}
	b.popControl()

	if s.Post != nil {
		if err := b.buildStatement(s.Post); err != nil {
			return err
		}
	}

	loopDepth, err := b.branchDepth(frameLoop)
	if err != nil {
		return err
	}
	b.emit(Instruction{Op: OpBr, LabelIdx: loopDepth})

	b.popControl() // loop
	b.popControl() // exit block
	return nil
}

// buildRangeStmt compiles an integer for range loop into WebAssembly block/loop instructions.
func (b *builder) buildRangeStmt(s *ast.RangeStmt) error {
	if s.Value != nil {
		return fmt.Errorf("range over integer permits only one iteration variable")
	}

	b.pushScope()
	defer b.popScope()

	key, err := b.resolveRangeKey(s)
	if err != nil {
		return err
	}
	rangeType, err := b.resolveRangeType(s.X, key)
	if err != nil {
		return err
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
	b.emitLocalSet(limitLocal)

	idxLocal := b.allocTempLocal("__range_idx", rangeType, wasmType)
	b.emit(b.zeroConstInstruction(wasmType))
	b.emitLocalSet(idxLocal)

	if key.defineKey {
		key.binding = b.declareLocal(key.ident.Name, rangeType, wasmType)
	}

	b.pushControl(frameLoopExit, OpBlock)
	b.pushControl(frameLoop, OpLoop)

	// Exit loop unless __range_idx < __range_limit.
	b.emitLocalGet(idxLocal)
	b.emitLocalGet(limitLocal)
	ltOp, err := binaryOpCode(token.LSS, rangeType)
	if err != nil {
		return err
	}
	exitDepth, err := b.branchDepth(frameLoopExit)
	if err != nil {
		return err
	}
	b.emit(Instruction{Op: ltOp}, Instruction{Op: OpI32Eqz}, Instruction{Op: OpBrIf, LabelIdx: exitDepth})

	if key.hasKey {
		b.emitLocalGet(idxLocal)
		b.emitLocalSet(key.binding)
	}

	b.pushControl(frameLoopCont, OpBlock)
	if err := b.buildStatement(s.Body); err != nil {
		return err
	}
	b.popControl()

	// Increment hidden loop counter by 1.
	b.emitLocalGet(idxLocal)
	oneInst, err := oneConstInstruction(rangeType)
	if err != nil {
		return err
	}
	addOp, err := binaryOpCode(token.ADD, rangeType)
	if err != nil {
		return err
	}
	b.emit(oneInst, Instruction{Op: addOp})
	b.emitLocalSet(idxLocal)

	loopDepth, err := b.branchDepth(frameLoop)
	if err != nil {
		return err
	}
	b.emit(Instruction{Op: OpBr, LabelIdx: loopDepth})

	b.popControl() // loop
	b.popControl() // exit block
	return nil
}

func (b *builder) resolveRangeKey(s *ast.RangeStmt) (rangeKeyInfo, error) {
	if s.Key == nil {
		return rangeKeyInfo{}, nil
	}
	ident, ok := s.Key.(*ast.Ident)
	if !ok {
		return rangeKeyInfo{}, fmt.Errorf("cannot assign to non-identifier: %T", s.Key)
	}

	switch s.Tok {
	case token.DEFINE:
		if ident.Name == "_" {
			return rangeKeyInfo{}, fmt.Errorf("no new variables on left side of :=")
		}
		return rangeKeyInfo{ident: ident, hasKey: true, defineKey: true}, nil
	case token.ASSIGN:
		if ident.Name == "_" {
			return rangeKeyInfo{ident: ident}, nil
		}
		binding, _, kind := b.lookupIdent(ident.Name)
		switch kind {
		case identNone:
			return rangeKeyInfo{}, fmt.Errorf("unknown identifier: %s", ident.Name)
		case identConst:
			return rangeKeyInfo{}, fmt.Errorf("cannot assign to constant: %s", ident.Name)
		default:
			return rangeKeyInfo{ident: ident, binding: binding, hasKey: true}, nil
		}
	default:
		return rangeKeyInfo{}, fmt.Errorf("unsupported range assignment operator: %s", s.Tok)
	}
}

func (b *builder) resolveRangeType(x ast.Expr, key rangeKeyInfo) (string, error) {
	xType := b.resolveExprType(x, "")
	if key.hasKey && !key.defineKey {
		if xType == "" {
			if !isIntegerType(key.binding.goType) {
				return "", fmt.Errorf("cannot use range type int32 as %s in assignment", key.binding.goType)
			}
			return key.binding.goType, nil
		}
		if !isIntegerType(xType) {
			return "", fmt.Errorf("cannot range over %s", xType)
		}
		if xType != key.binding.goType {
			return "", fmt.Errorf("cannot use range type %s as %s in assignment", xType, key.binding.goType)
		}
		return xType, nil
	}
	if xType == "" {
		xType = "int32"
	}
	if !isIntegerType(xType) {
		return "", fmt.Errorf("cannot range over %s", xType)
	}
	return xType, nil
}

// buildBranchStmt compiles break and continue statements.
func (b *builder) buildBranchStmt(s *ast.BranchStmt) error {
	switch s.Tok {
	case token.BREAK:
		depth, err := b.branchDepth(frameLoopExit)
		if err != nil {
			return fmt.Errorf("break is not in a loop")
		}
		b.emit(Instruction{Op: OpBr, LabelIdx: depth})
		return nil
	case token.CONTINUE:
		depth, err := b.branchDepth(frameLoopCont)
		if err != nil {
			return fmt.Errorf("continue is not in a loop")
		}
		b.emit(Instruction{Op: OpBr, LabelIdx: depth})
		return nil
	default:
		return fmt.Errorf("unsupported branch statement: %v", s.Tok)
	}
}
