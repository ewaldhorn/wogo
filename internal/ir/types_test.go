package ir

import (
	"strings"
	"testing"
)

// ------------------------------------------------------------------------------------------------
// Ensure we can parse Go types to Wasm types
func TestParseGoType(t *testing.T) {
	tests := []struct {
		name    string
		goType  string
		want    Type
		wantErr string
	}{
		{"int32", "int32", TypeI32, ""},
		{"int", "int", TypeI32, ""},
		{"uint32", "uint32", TypeI32, ""},
		{"uint", "uint", TypeI32, ""},
		{"bool", "bool", TypeI32, ""},
		{"int64", "int64", TypeI64, ""},
		{"uint64", "uint64", TypeI64, ""},
		{"float32", "float32", TypeF32, ""},
		{"float64", "float64", TypeF64, ""},
		{"string rejected", "string", 0, `unsupported type "string"`},
		{"empty rejected", "", 0, `unsupported type ""`},
		{"narrow int rejected", "int16", 0, `unsupported type "int16"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGoType(tt.goType)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ParseGoType(%q) unexpected error: %v", tt.goType, err)
				}
				if got != tt.want {
					t.Errorf("ParseGoType(%q) = %v, want %v", tt.goType, got, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("ParseGoType(%q) = %v, want error %q", tt.goType, got, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ParseGoType(%q) error = %q, want it to contain %q", tt.goType, err.Error(), tt.wantErr)
			}
		})
	}
}

// ------------------------------------------------------------------------------------------------
// Sanity check parsing instructions to WAT string expressions
func TestInstruction_WatString(t *testing.T) {
	tests := []struct {
		name string
		inst Instruction
		want string
	}{
		{"local.get named", Instruction{Op: OpLocalGet, LocalIdx: 0, LocalName: "a"}, "local.get $a"},
		{"local.get unnamed", Instruction{Op: OpLocalGet, LocalIdx: 2}, "local.get 2"},
		{"i32.const raw", Instruction{Op: OpI32Const, I32: 42, Raw: "42"}, "i32.const 42"},
		{"i32.const no raw", Instruction{Op: OpI32Const, I32: -3}, "i32.const -3"},
		{"i64.const raw", Instruction{Op: OpI64Const, I64: 9, Raw: "9"}, "i64.const 9"},
		{"i64.const no raw", Instruction{Op: OpI64Const, I64: 0}, "i64.const 0"},
		{"f32.const int raw", Instruction{Op: OpF32Const, F32: 1, Raw: "1"}, "f32.const 1.0"},
		{"f32.const float raw", Instruction{Op: OpF32Const, F32: 2.5, Raw: "2.5"}, "f32.const 2.5"},
		{"f32.const no raw", Instruction{Op: OpF32Const, F32: 2.5}, "f32.const 2.5"},
		{"f32.const whole no raw", Instruction{Op: OpF32Const, F32: 3}, "f32.const 3.0"},
		{"f64.const int raw", Instruction{Op: OpF64Const, F64: 1, Raw: "1"}, "f64.const 1.0"},
		{"f64.const float raw", Instruction{Op: OpF64Const, Raw: "3.141592653589793"}, "f64.const 3.141592653589793"},
		{"f64.const no raw", Instruction{Op: OpF64Const, F64: 0.5}, "f64.const 0.5"},
		{"i32.add", Instruction{Op: OpI32Add}, "i32.add"},
		{"i32.sub", Instruction{Op: OpI32Sub}, "i32.sub"},
		{"i32.mul", Instruction{Op: OpI32Mul}, "i32.mul"},
		{"i32.div_s", Instruction{Op: OpI32DivS}, "i32.div_s"},
		{"i32.div_u", Instruction{Op: OpI32DivU}, "i32.div_u"},
		{"i32.rem_s", Instruction{Op: OpI32RemS}, "i32.rem_s"},
		{"i32.rem_u", Instruction{Op: OpI32RemU}, "i32.rem_u"},
		{"i32.eqz", Instruction{Op: OpI32Eqz}, "i32.eqz"},
		{"i64.add", Instruction{Op: OpI64Add}, "i64.add"},
		{"i64.sub", Instruction{Op: OpI64Sub}, "i64.sub"},
		{"i64.mul", Instruction{Op: OpI64Mul}, "i64.mul"},
		{"i64.div_s", Instruction{Op: OpI64DivS}, "i64.div_s"},
		{"i64.div_u", Instruction{Op: OpI64DivU}, "i64.div_u"},
		{"i64.rem_s", Instruction{Op: OpI64RemS}, "i64.rem_s"},
		{"i64.rem_u", Instruction{Op: OpI64RemU}, "i64.rem_u"},
		{"i64.eqz", Instruction{Op: OpI64Eqz}, "i64.eqz"},
		{"f32.neg", Instruction{Op: OpF32Neg}, "f32.neg"},
		{"f32.add", Instruction{Op: OpF32Add}, "f32.add"},
		{"f32.sub", Instruction{Op: OpF32Sub}, "f32.sub"},
		{"f32.mul", Instruction{Op: OpF32Mul}, "f32.mul"},
		{"f32.div", Instruction{Op: OpF32Div}, "f32.div"},
		{"f64.neg", Instruction{Op: OpF64Neg}, "f64.neg"},
		{"f64.add", Instruction{Op: OpF64Add}, "f64.add"},
		{"f64.sub", Instruction{Op: OpF64Sub}, "f64.sub"},
		{"f64.mul", Instruction{Op: OpF64Mul}, "f64.mul"},
		{"f64.div", Instruction{Op: OpF64Div}, "f64.div"},
		{"return", Instruction{Op: OpReturn}, "return"},
		{"unrendered op is empty", Instruction{Op: OpEnd}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.inst.WatString(); got != tt.want {
				t.Errorf("WatString() = %q, want %q", got, tt.want)
			}
		})
	}
}
