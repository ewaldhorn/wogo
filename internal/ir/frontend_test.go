package ir

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

// ------------------------------------------------------------------------------------------------
// Sanity checker to make sure we can do basic parsing, at least.
// If this smoke test fails, there are big problems.
func TestBuildModule_Simple(t *testing.T) {
	src := `package calc

const PI = 3.141592653589793

func DegreesToRads(deg float64) float64 {
	const LocalConst = 180.0
	return deg * (PI / LocalConst)
}
`
	mod, err := BuildModule(src)
	if err != nil {
		t.Fatalf("BuildModule failed: %v", err)
	}

	if len(mod.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(mod.Functions))
	}

	fn := mod.Functions[0]
	if fn.Name != "DegreesToRads" {
		t.Errorf("expected function name DegreesToRads, got %s", fn.Name)
	}

	if len(fn.Params) != 1 || fn.Params[0].Name != "deg" || fn.Params[0].Type != TypeF64 {
		t.Errorf("unexpected params: %+v", fn.Params)
	}

	if len(fn.Results) != 1 || fn.Results[0] != TypeF64 {
		t.Errorf("unexpected results: %+v", fn.Results)
	}

	if len(fn.Instructions) == 0 {
		t.Fatalf("expected instructions, got none")
	}

	last := fn.Instructions[len(fn.Instructions)-1]
	if last.Op != OpReturn {
		t.Errorf("expected last instruction to be OpReturn, got %v", last.Op)
	}
}

// ------------------------------------------------------------------------------------------------
func TestBuildModule_Invalid(t *testing.T) {
	_, err := BuildModule("broken syntax {")
	if err == nil {
		t.Error("expected syntax error, got nil")
	}
}

// ------------------------------------------------------------------------------------------------
func TestBuildModule_GoInt(t *testing.T) {
	mod, err := BuildModule("package calc\n\nfunc Add(a int, b int) int { return a + b }\n")
	if err != nil {
		t.Fatalf("expected int to compile, got %v", err)
	}

	fn := mod.Functions[0]
	for _, p := range fn.Params {
		if p.Type != TypeI32 {
			t.Errorf("param %s: expected TypeI32, got %v", p.Name, p.Type)
		}
	}
	if len(fn.Results) != 1 || fn.Results[0] != TypeI32 {
		t.Errorf("unexpected results: %+v", fn.Results)
	}
}

// ------------------------------------------------------------------------------------------------
// watLines compiles src and renders one function's instructions as WAT lines.
func watLines(t *testing.T, src string, funcIdx int) []string {
	t.Helper()

	mod, err := BuildModule(src)
	if err != nil {
		t.Fatalf("BuildModule failed: %v", err)
	}
	if funcIdx >= len(mod.Functions) {
		t.Fatalf("want function index %d, module has %d functions", funcIdx, len(mod.Functions))
	}

	var lines []string
	for _, inst := range mod.Functions[funcIdx].Instructions {
		lines = append(lines, inst.WatString())
	}
	return lines
}

// ------------------------------------------------------------------------------------------------
func TestBuildFunction_Lowering(t *testing.T) {
	preamble := "package calc\n\n"
	tests := []struct {
		name string
		fn   string
		want []string
	}{
		{"add i32",
			"func F(a int32, b int32) int32 { return a + b }",
			[]string{"local.get $a", "local.get $b", "i32.add", "return"}},
		{"sub i32",
			"func F(a int32, b int32) int32 { return a - b }",
			[]string{"local.get $a", "local.get $b", "i32.sub", "return"}},
		{"mul i32",
			"func F(a int32, b int32) int32 { return a * b }",
			[]string{"local.get $a", "local.get $b", "i32.mul", "return"}},
		{"div_s i32",
			"func F(a int32, b int32) int32 { return a / b }",
			[]string{"local.get $a", "local.get $b", "i32.div_s", "return"}},
		{"rem_s i32",
			"func F(a int32, b int32) int32 { return a % b }",
			[]string{"local.get $a", "local.get $b", "i32.rem_s", "return"}},
		{"div_u uint32",
			"func F(a uint32, b uint32) uint32 { return a / b }",
			[]string{"local.get $a", "local.get $b", "i32.div_u", "return"}},
		{"rem_u uint32",
			"func F(a uint32, b uint32) uint32 { return a % b }",
			[]string{"local.get $a", "local.get $b", "i32.rem_u", "return"}},
		{"div_s bare int",
			"func F(a int, b int) int { return a / b }",
			[]string{"local.get $a", "local.get $b", "i32.div_s", "return"}},
		{"div_u bare uint",
			"func F(a uint, b uint) uint { return a / b }",
			[]string{"local.get $a", "local.get $b", "i32.div_u", "return"}},
		{"add i64",
			"func F(a int64, b int64) int64 { return a + b }",
			[]string{"local.get $a", "local.get $b", "i64.add", "return"}},
		{"div_s i64",
			"func F(a int64, b int64) int64 { return a / b }",
			[]string{"local.get $a", "local.get $b", "i64.div_s", "return"}},
		{"div_u uint64",
			"func F(a uint64, b uint64) uint64 { return a / b }",
			[]string{"local.get $a", "local.get $b", "i64.div_u", "return"}},
		{"rem_u uint64",
			"func F(a uint64, b uint64) uint64 { return a % b }",
			[]string{"local.get $a", "local.get $b", "i64.rem_u", "return"}},
		{"add f32",
			"func F(a float32, b float32) float32 { return a + b }",
			[]string{"local.get $a", "local.get $b", "f32.add", "return"}},
		{"div f32",
			"func F(a float32, b float32) float32 { return a / b }",
			[]string{"local.get $a", "local.get $b", "f32.div", "return"}},
		{"add f64",
			"func F(a float64, b float64) float64 { return a + b }",
			[]string{"local.get $a", "local.get $b", "f64.add", "return"}},
		{"div f64",
			"func F(a float64, b float64) float64 { return a / b }",
			[]string{"local.get $a", "local.get $b", "f64.div", "return"}},
		{"neg f32",
			"func F(a float32) float32 { return -a }",
			[]string{"local.get $a", "f32.neg", "return"}},
		{"neg f64",
			"func F(a float64) float64 { return -a }",
			[]string{"local.get $a", "f64.neg", "return"}},
		{"neg i32",
			"func F(a int32) int32 { return -a }",
			[]string{"i32.const 0", "local.get $a", "i32.sub", "return"}},
		{"neg i64",
			"func F(a int64) int64 { return -a }",
			[]string{"i64.const 0", "local.get $a", "i64.sub", "return"}},
		{"not bool",
			"func F(a bool) bool { return !a }",
			[]string{"local.get $a", "i32.eqz", "return"}},
		{"not i64",
			"func F(a int64) int64 { return !a }",
			[]string{"local.get $a", "i32.eqz", "return"}},
		{"not chain",
			"func F() bool { return !!true }",
			[]string{"i32.const 1", "i32.eqz", "i32.eqz", "return"}},
		{"bool literal true",
			"func F() bool { return true }",
			[]string{"i32.const 1", "return"}},
		{"bool literal false",
			"func F() bool { return false }",
			[]string{"i32.const 0", "return"}},
		{"precedence mul over add",
			"func F(a int32, b int32, c int32) int32 { return a + b * c }",
			[]string{"local.get $a", "local.get $b", "local.get $c", "i32.mul", "i32.add", "return"}},
		{"parens override precedence",
			"func F(a int32, b int32, c int32) int32 { return (a + b) * c }",
			[]string{"local.get $a", "local.get $b", "i32.add", "local.get $c", "i32.mul", "return"}},
		{"int literal",
			"func F() int32 { return 42 }",
			[]string{"i32.const 42", "return"}},
		{"int literal in f64 context",
			"func F() float64 { return 1 }",
			[]string{"f64.const 1.0", "return"}},
		{"package const",
			"const Ten = 10\n\nfunc F(a int32) int32 { return a + Ten }",
			[]string{"local.get $a", "i32.const 10", "i32.add", "return"}},
		{"local const",
			"func F(a int32) int32 { const Ten = 10; return a + Ten }",
			[]string{"local.get $a", "i32.const 10", "i32.add", "return"}},
		{"typed const",
			"func F(a int32) int32 { const X int32 = 7; return a + X }",
			[]string{"local.get $a", "i32.const 7", "i32.add", "return"}},
		{"math.Pi selector",
			"import \"math\"\n\nfunc F() float64 { return math.Pi }",
			[]string{"f64.const 3.141592653589793", "return"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := watLines(t, preamble+tt.fn+"\n", 0)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("instructions mismatch:\n got:\n%s\nwant:\n%s",
					strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

// ------------------------------------------------------------------------------------------------
func TestBuildModule_MultiFunction(t *testing.T) {
	src := "package calc\n\n" +
		"func Double(a int32) int32 { return a + a }\n" +
		"func Neg(a int32) int32 { return -a }\n"

	mod, err := BuildModule(src)
	if err != nil {
		t.Fatalf("BuildModule failed: %v", err)
	}
	if len(mod.Functions) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(mod.Functions))
	}

	want := map[string][]string{
		"Double": {"local.get $a", "local.get $a", "i32.add", "return"},
		"Neg":    {"i32.const 0", "local.get $a", "i32.sub", "return"},
	}
	for _, fn := range mod.Functions {
		var got []string
		for _, inst := range fn.Instructions {
			got = append(got, inst.WatString())
		}
		if !reflect.DeepEqual(got, want[fn.Name]) {
			t.Errorf("%s mismatch:\n got:\n%s\nwant:\n%s",
				fn.Name, strings.Join(got, "\n"), strings.Join(want[fn.Name], "\n"))
		}
	}
}

// ------------------------------------------------------------------------------------------------
func TestBuildModule_UnsupportedType(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			"string param",
			"package calc\n\nfunc Bad(a string) string { return a }\n",
			`unsupported type "string"`,
		},
		{
			"string result",
			"package calc\n\nfunc Bad() string { return \"hi\" }\n",
			`unsupported type "string"`,
		},
		{
			"string const",
			"package calc\n\nconst x string = \"hi\"\n\nfunc F() int32 { return x }\n",
			`unsupported type "string"`,
		},
		{
			"pointer param",
			"package calc\n\nfunc F(a *int32) int32 { return a }\n",
			"unsupported type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildModule(tt.src)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// ------------------------------------------------------------------------------------------------
func TestBuildModule_Errors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			"unknown identifier",
			"package calc\n\nfunc F(a int32) int32 { return b }\n",
			"unknown identifier: b",
		},
		{
			"for statement unsupported",
			"package calc\n\nfunc F(a int32) int32 { for {}; return a }\n",
			"unsupported statement type",
		},
		{
			"if statement unsupported",
			"package calc\n\nfunc F(a int32) int32 { if a == 1 { return a }; return a }\n",
			"unsupported statement type",
		},
		{
			"type declaration unsupported",
			"package calc\n\nfunc F(a int32) int32 { type MyInt int32; return a }\n",
			"unsupported declaration statement",
		},
		{
			"call expression unsupported",
			"package calc\n\nfunc F(a int32) int32 { return g(a) }\n",
			"unsupported expression type",
		},
		{
			"comparison operator unsupported",
			"package calc\n\nfunc F(a int32, b int32) bool { return a == b }\n",
			"unsupported binary operator",
		},
		{
			"unary xor unsupported",
			"package calc\n\nfunc F(a int32) int32 { return ^a }\n",
			"unsupported unary operator",
		},
		{
			"imaginary literal unsupported",
			"package calc\n\nfunc F() float64 { return 1i }\n",
			"unsupported literal kind",
		},
		{
			"redefine with := rejected",
			"package calc\n\nfunc F() int32 { x := 1; x := 2; return x }\n",
			"no new variables on left side of :=",
		},
		{
			"repeated name in := rejected",
			"package calc\n\nfunc F() int32 { x, x := 1, 2; return x }\n",
			"repeated on left side of :=",
		},
		{
			"self-referential := rejected",
			"package calc\n\nfunc F() int32 { x := x + 1; return x }\n",
			"unknown identifier: x",
		},
		{
			"var redeclare rejected",
			"package calc\n\nfunc F() int32 { var x int32; var x int32; return x }\n",
			"x redeclared in this block",
		},
		{
			"var shadowing param rejected",
			"package calc\n\nfunc F(x int32) int32 { var x int32; return x }\n",
			"x redeclared in this block",
		},
		{
			"self-referential var rejected",
			"package calc\n\nfunc F() int32 { var x int32 = x + 1; return x }\n",
			"unknown identifier: x",
		},
		{
			"assign to unknown identifier",
			"package calc\n\nfunc F() int32 { y = 1; return 0 }\n",
			"unknown identifier: y",
		},
		{
			"short var count mismatch",
			"package calc\n\nfunc F() int32 { x, y := 1; return x }\n",
			"assignment mismatch",
		},
		{
			"compound assign to unknown identifier",
			"package calc\n\nfunc F() int32 { x += 1; return 0 }\n",
			"unknown identifier: x",
		},
		{
			"bitwise assign unsupported",
			"package calc\n\nfunc F() int32 { x := 1; x &= 2; return x }\n",
			"unsupported assignment operator",
		},
		{
			"non-identifier assign target",
			"package calc\n\nfunc F() int32 { 1 = 2; return 0 }\n",
			"cannot assign to non-identifier",
		},
		{
			"var too many values",
			"package calc\n\nfunc F() int32 { var x int32 = 1, 2; return x }\n",
			"assignment mismatch",
		},
		{
			"var too few values",
			"package calc\n\nfunc F() int32 { var x, y int32 = 1; return x }\n",
			"assignment mismatch",
		},
		{
			"cross-referential := rejected",
			"package calc\n\nfunc F() int32 { x, y := 1, x; return y }\n",
			"unknown identifier: x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildModule(tt.src)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// ------------------------------------------------------------------------------------------------
func TestBinaryOpCode(t *testing.T) {
	tests := []struct {
		name     string
		op       token.Token
		exprType string
		want     OpCode
		wantErr  string
	}{
		{"add i32", token.ADD, "int32", OpI32Add, ""},
		{"quo i32", token.QUO, "int32", OpI32DivS, ""},
		{"rem int", token.REM, "int", OpI32RemS, ""},
		{"quo uint32", token.QUO, "uint32", OpI32DivU, ""},
		{"rem uint", token.REM, "uint", OpI32RemU, ""},
		{"mul i64", token.MUL, "int64", OpI64Mul, ""},
		{"quo i64", token.QUO, "int64", OpI64DivS, ""},
		{"quo uint64", token.QUO, "uint64", OpI64DivU, ""},
		{"add f32", token.ADD, "float32", OpF32Add, ""},
		{"quo f32", token.QUO, "float32", OpF32Div, ""},
		{"sub f64", token.SUB, "float64", OpF64Sub, ""},
		{"eql i32 rejected", token.EQL, "int32", 0, "unsupported binary operator"},
		{"land bool rejected", token.LAND, "bool", 0, "unsupported binary operator"},
		{"add string rejected", token.ADD, "string", 0, `unsupported type "string"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := binaryOpCode(tt.op, tt.exprType)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("binaryOpCode(%s, %q) unexpected error: %v", tt.op, tt.exprType, err)
				}
				if got != tt.want {
					t.Errorf("binaryOpCode(%s, %q) = %v, want %v", tt.op, tt.exprType, got, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("binaryOpCode(%s, %q) = %v, want error %q", tt.op, tt.exprType, got, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// ------------------------------------------------------------------------------------------------
func mustParseExpr(t *testing.T, src string) ast.Expr {
	t.Helper()

	expr, err := goparser.ParseExpr(src)
	if err != nil {
		t.Fatalf("ParseExpr(%q) failed: %v", src, err)
	}
	return expr
}

// ------------------------------------------------------------------------------------------------
func TestResolveExprType(t *testing.T) {
	b := &builder{
		packageConstants: map[string]constantDef{
			"PI": {expr: mustParseExpr(t, "3.14")},
		},
		localConstants: map[string]constantDef{
			"C": {expr: mustParseExpr(t, "10")},
			"T": {explicitType: "int64", expr: mustParseExpr(t, "1")},
		},
		localTypes: map[string]string{
			"a": "int32",
			"f": "float64",
		},
	}

	tests := []struct {
		name   string
		expr   string
		target string
		want   string
	}{
		{"bool literal", "true", "int32", "bool"},
		{"param type", "a", "float64", "int32"},
		{"unknown ident falls back to target", "zzz", "float32", "float32"},
		{"untyped const resolves from value", "C", "float64", "float64"},
		{"typed const uses explicit type", "T", "int32", "int64"},
		{"package const float", "PI", "int32", "float64"},
		{"float literal f32 target", "1.5", "float32", "float32"},
		{"float literal other target", "1.5", "int32", "float64"},
		{"int literal uses target", "42", "uint32", "uint32"},
		{"binary prefers non-i32 side", "a + f", "int32", "float64"},
		{"binary all i32", "a + a", "float64", "int32"},
		{"parens transparent", "(f)", "int32", "float64"},
		{"not is bool", "!a", "int32", "bool"},
		{"neg keeps operand type", "-f", "int32", "float64"},
		{"math.Pi is f64", "math.Pi", "int32", "float64"},
		{"unknown selector falls back to target", "math.Nope", "int32", "int32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.resolveExprType(mustParseExpr(t, tt.expr), tt.target)
			if got != tt.want {
				t.Errorf("resolveExprType(%q, %q) = %q, want %q", tt.expr, tt.target, got, tt.want)
			}
		})
	}
}

// ------------------------------------------------------------------------------------------------
func TestBuildFunction_LocalVariables(t *testing.T) {
	src := `package test

func Calc(a int32) int32 {
	x := 10
	x = x + a
	x += 5
	var y int32 = 2
	var z int32
	return x + y + z
}
`
	mod, err := BuildModule(src)
	if err != nil {
		t.Fatalf("BuildModule failed: %v", err)
	}

	if len(mod.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(mod.Functions))
	}

	fn := mod.Functions[0]
	if len(fn.Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(fn.Params))
	}
	if len(fn.Locals) != 3 {
		t.Fatalf("expected 3 locals, got %d: %+v", len(fn.Locals), fn.Locals)
	}

	if fn.Locals[0].Name != "x" || fn.Locals[0].Type != TypeI32 {
		t.Errorf("unexpected local 0: %+v", fn.Locals[0])
	}
	if fn.Locals[1].Name != "y" || fn.Locals[1].Type != TypeI32 {
		t.Errorf("unexpected local 1: %+v", fn.Locals[1])
	}
	if fn.Locals[2].Name != "z" || fn.Locals[2].Type != TypeI32 {
		t.Errorf("unexpected local 2: %+v", fn.Locals[2])
	}
}

// ------------------------------------------------------------------------------------------------
func TestBuildFunction_LocalVariables_Lowering(t *testing.T) {
	src := "package calc\n\nfunc F(a int32) int32 { x := a + 1; x += 2; return x }\n"
	want := []string{
		"local.get $a", "i32.const 1", "i32.add", "local.set $x",
		"local.get $x", "i32.const 2", "i32.add", "local.set $x",
		"local.get $x", "return",
	}
	got := watLines(t, src, 0)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("locals lowering mismatch:\n got:\n%s\nwant:\n%s",
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// ------------------------------------------------------------------------------------------------
// Mixed := reuses the existing variable and declares only the new one,
// matching Go (which requires at least one new variable on the left).
func TestBuildFunction_ShortVarMixed(t *testing.T) {
	src := "package calc\n\nfunc F(a int32) int32 { x := a; x, y := x + 1, 2; return x + y }\n"
	mod, err := BuildModule(src)
	if err != nil {
		t.Fatalf("BuildModule failed: %v", err)
	}

	fn := mod.Functions[0]
	if len(fn.Locals) != 2 {
		t.Fatalf("expected 2 locals, got %d: %+v", len(fn.Locals), fn.Locals)
	}
	if fn.Locals[0].Name != "x" || fn.Locals[1].Name != "y" {
		t.Errorf("unexpected locals: %+v", fn.Locals)
	}

	want := []string{
		"local.get $a", "local.set $x",
		"local.get $x", "i32.const 1", "i32.add", "i32.const 2", "local.set $y", "local.set $x",
		"local.get $x", "local.get $y", "i32.add", "return",
	}
	var got []string
	for _, inst := range fn.Instructions {
		got = append(got, inst.WatString())
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("mixed := lowering mismatch:\n got:\n%s\nwant:\n%s",
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// ------------------------------------------------------------------------------------------------
// Multi-assign evaluates every value before storing any of them, so aliased
// targets like a swap observe pre-assignment state.
func TestBuildFunction_MultiAssignSwap(t *testing.T) {
	src := "package calc\n\nfunc F(a int32, b int32) int32 { a, b = b, a; return a }\n"
	want := []string{
		"local.get $b", "local.get $a", "local.set $b", "local.set $a",
		"local.get $a", "return",
	}
	got := watLines(t, src, 0)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("swap lowering mismatch:\n got:\n%s\nwant:\n%s",
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// ------------------------------------------------------------------------------------------------
// A repeated = target keeps its last value, like Go's left-to-right
// assignment; earlier duplicates only discard their value.
func TestBuildFunction_MultiAssignDuplicateTarget(t *testing.T) {
	src := "package calc\n\nfunc F() int32 { x := 0; x, x = 1, 2; return x }\n"
	want := []string{
		"i32.const 0", "local.set $x",
		"i32.const 1", "i32.const 2", "local.set $x", "drop",
		"local.get $x", "return",
	}
	got := watLines(t, src, 0)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("duplicate target lowering mismatch:\n got:\n%s\nwant:\n%s",
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// ------------------------------------------------------------------------------------------------
// Without an explicit type, each var name takes the type of its own value.
func TestBuildFunction_VarPerValueInference(t *testing.T) {
	src := "package calc\n\nfunc F() float64 { var a, b = 1, 2.5; return b }\n"
	mod, err := BuildModule(src)
	if err != nil {
		t.Fatalf("BuildModule failed: %v", err)
	}

	fn := mod.Functions[0]
	if len(fn.Locals) != 2 {
		t.Fatalf("expected 2 locals, got %d: %+v", len(fn.Locals), fn.Locals)
	}
	if fn.Locals[0].Name != "a" || fn.Locals[0].Type != TypeI32 {
		t.Errorf("unexpected local 0: %+v", fn.Locals[0])
	}
	if fn.Locals[1].Name != "b" || fn.Locals[1].Type != TypeF64 {
		t.Errorf("unexpected local 1: %+v", fn.Locals[1])
	}
}
