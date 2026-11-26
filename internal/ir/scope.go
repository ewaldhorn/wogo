package ir

import (
	"fmt"
	"go/ast"
	"maps"
	"slices"
)

type constantDef struct {
	explicitType string
	expr         ast.Expr
	env          map[string]constantDef
}

type localBinding struct {
	idx      uint32
	wasmName string
	goType   string
}

type identKind int

const (
	identNone identKind = iota
	identVar
	identConst
)

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

func (b *builder) lookupIdent(name string) (localBinding, constantDef, identKind) {
	if b.constEnv != nil {
		if cDef, ok := b.constEnv[name]; ok {
			return localBinding{}, cDef, identConst
		}
		return localBinding{}, constantDef{}, identNone
	}
	for _, sc := range slices.Backward(b.scopes) {
		if v, ok := sc.vars[name]; ok {
			return v, constantDef{}, identVar
		}
		if c, ok := sc.consts[name]; ok {
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
	maps.Copy(env, b.packageConstants)
	for _, sc := range b.scopes {
		for k := range sc.vars {
			delete(env, k)
		}
		maps.Copy(env, sc.consts)
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
