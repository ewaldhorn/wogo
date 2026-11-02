package ir

import (
	"fmt"
	"strconv"
	"strings"
)

// ------------------------------------------------------------------------------------------------
// WebAssembly value types.
type Type byte

const (
	TypeI32 Type = 0x7F
	TypeI64 Type = 0x7E
	TypeF32 Type = 0x7D
	TypeF64 Type = 0x7C
)

func (t Type) String() string {
	switch t {
	case TypeI32:
		return "i32"
	case TypeI64:
		return "i64"
	case TypeF32:
		return "f32"
	case TypeF64:
		return "f64"
	default:
		return "i32"
	}
}

// ------------------------------------------------------------------------------------------------
// ParseGoType maps a Go type name to a WebAssembly value Type.
// Go's int and uint are explicit i32 and anything else now errors correctly, I hope.
func ParseGoType(name string) (Type, error) {
	switch name {
	case "int32", "int", "uint32", "uint", "bool":
		return TypeI32, nil
	case "int64", "uint64":
		return TypeI64, nil
	case "float32":
		return TypeF32, nil
	case "float64":
		return TypeF64, nil
	default:
		return 0, fmt.Errorf("unsupported type %q", name)
	}
}

// ------------------------------------------------------------------------------------------------
// Instruction represents a single low-level WebAssembly instruction in the IR.
type Instruction struct {
	Op        OpCode
	LocalIdx  uint32
	LocalName string
	I32       int32
	I64       int64
	F32       float32
	F64       float64
	Raw       string
}

// ------------------------------------------------------------------------------------------------
// WatString writes the instructions as a WAT string
func (inst Instruction) WatString() string {
	switch inst.Op {
	case OpLocalGet:
		if inst.LocalName != "" {
			return fmt.Sprintf("local.get $%s", inst.LocalName)
		}
		return fmt.Sprintf("local.get %d", inst.LocalIdx)
	case OpLocalSet:
		if inst.LocalName != "" {
			return fmt.Sprintf("local.set $%s", inst.LocalName)
		}
		return fmt.Sprintf("local.set %d", inst.LocalIdx)
	case OpI32Const:
		if inst.Raw != "" {
			return fmt.Sprintf("i32.const %s", inst.Raw)
		}
		return fmt.Sprintf("i32.const %d", inst.I32)
	case OpI64Const:
		if inst.Raw != "" {
			return fmt.Sprintf("i64.const %s", inst.Raw)
		}
		return fmt.Sprintf("i64.const %d", inst.I64)
	case OpF32Const:
		if inst.Raw != "" {
			raw := inst.Raw
			if !strings.Contains(raw, ".") && !strings.Contains(raw, "e") {
				raw += ".0"
			}
			return fmt.Sprintf("f32.const %s", raw)
		}
		return fmt.Sprintf("f32.const %s", formatFloat(float64(inst.F32)))
	case OpF64Const:
		if inst.Raw != "" {
			raw := inst.Raw
			if !strings.Contains(raw, ".") && !strings.Contains(raw, "e") {
				raw += ".0"
			}
			return fmt.Sprintf("f64.const %s", raw)
		}
		return fmt.Sprintf("f64.const %s", formatFloat(inst.F64))
	case OpI32Add:
		return "i32.add"
	case OpI32Sub:
		return "i32.sub"
	case OpI32Mul:
		return "i32.mul"
	case OpI32DivS:
		return "i32.div_s"
	case OpI32DivU:
		return "i32.div_u"
	case OpI32RemS:
		return "i32.rem_s"
	case OpI32RemU:
		return "i32.rem_u"
	case OpI32Eqz:
		return "i32.eqz"
	case OpI32Eq:
		return "i32.eq"
	case OpI32Ne:
		return "i32.ne"
	case OpI32LtS:
		return "i32.lt_s"
	case OpI32LtU:
		return "i32.lt_u"
	case OpI32GtS:
		return "i32.gt_s"
	case OpI32GtU:
		return "i32.gt_u"
	case OpI32LeS:
		return "i32.le_s"
	case OpI32LeU:
		return "i32.le_u"
	case OpI32GeS:
		return "i32.ge_s"
	case OpI32GeU:
		return "i32.ge_u"
	case OpI64Add:
		return "i64.add"
	case OpI64Sub:
		return "i64.sub"
	case OpI64Mul:
		return "i64.mul"
	case OpI64DivS:
		return "i64.div_s"
	case OpI64DivU:
		return "i64.div_u"
	case OpI64RemS:
		return "i64.rem_s"
	case OpI64RemU:
		return "i64.rem_u"
	case OpI64Eqz:
		return "i64.eqz"
	case OpI64Eq:
		return "i64.eq"
	case OpI64Ne:
		return "i64.ne"
	case OpI64LtS:
		return "i64.lt_s"
	case OpI64LtU:
		return "i64.lt_u"
	case OpI64GtS:
		return "i64.gt_s"
	case OpI64GtU:
		return "i64.gt_u"
	case OpI64LeS:
		return "i64.le_s"
	case OpI64LeU:
		return "i64.le_u"
	case OpI64GeS:
		return "i64.ge_s"
	case OpI64GeU:
		return "i64.ge_u"
	case OpF32Neg:
		return "f32.neg"
	case OpF32Add:
		return "f32.add"
	case OpF32Sub:
		return "f32.sub"
	case OpF32Mul:
		return "f32.mul"
	case OpF32Div:
		return "f32.div"
	case OpF32Eq:
		return "f32.eq"
	case OpF32Ne:
		return "f32.ne"
	case OpF32Lt:
		return "f32.lt"
	case OpF32Gt:
		return "f32.gt"
	case OpF32Le:
		return "f32.le"
	case OpF32Ge:
		return "f32.ge"
	case OpF64Neg:
		return "f64.neg"
	case OpF64Add:
		return "f64.add"
	case OpF64Sub:
		return "f64.sub"
	case OpF64Mul:
		return "f64.mul"
	case OpF64Div:
		return "f64.div"
	case OpF64Eq:
		return "f64.eq"
	case OpF64Ne:
		return "f64.ne"
	case OpF64Lt:
		return "f64.lt"
	case OpF64Gt:
		return "f64.gt"
	case OpF64Le:
		return "f64.le"
	case OpF64Ge:
		return "f64.ge"
	case OpDrop:
		return "drop"
	case OpReturn:
		return "return"
	default:
		return ""
	}
}

// ------------------------------------------------------------------------------------------------
// formatFloat converts a float64 into a string, handy for constants in WAT format
func formatFloat(val float64) string {
	s := strconv.FormatFloat(val, 'g', -1, 64)
	if !strings.Contains(s, ".") && !strings.Contains(s, "e") {
		s += ".0"
	}
	return s
}
