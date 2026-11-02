package ir

// ------------------------------------------------------------------------------------------------
// WebAssembly opcodes.
// There's lots more, but I don't use all of them.
type OpCode byte

const (
	// Control Flow
	OpEnd      OpCode = 0x0B
	OpReturn   OpCode = 0x0F
	OpDrop     OpCode = 0x1A
	OpLocalGet OpCode = 0x20
	OpLocalSet OpCode = 0x21

	// Constants
	OpI32Const OpCode = 0x41
	OpI64Const OpCode = 0x42
	OpF32Const OpCode = 0x43
	OpF64Const OpCode = 0x44

	// Comparisons - i32
	OpI32Eqz OpCode = 0x45
	OpI32Eq  OpCode = 0x46
	OpI32Ne  OpCode = 0x47
	OpI32LtS OpCode = 0x48
	OpI32LtU OpCode = 0x49
	OpI32GtS OpCode = 0x4A
	OpI32GtU OpCode = 0x4B
	OpI32LeS OpCode = 0x4C
	OpI32LeU OpCode = 0x4D
	OpI32GeS OpCode = 0x4E
	OpI32GeU OpCode = 0x4F

	// Comparisons - i64
	OpI64Eqz OpCode = 0x50
	OpI64Eq  OpCode = 0x51
	OpI64Ne  OpCode = 0x52
	OpI64LtS OpCode = 0x53
	OpI64LtU OpCode = 0x54
	OpI64GtS OpCode = 0x55
	OpI64GtU OpCode = 0x56
	OpI64LeS OpCode = 0x57
	OpI64LeU OpCode = 0x58
	OpI64GeS OpCode = 0x59
	OpI64GeU OpCode = 0x5A

	// Comparisons - f32
	OpF32Eq OpCode = 0x5B
	OpF32Ne OpCode = 0x5C
	OpF32Lt OpCode = 0x5D
	OpF32Gt OpCode = 0x5E
	OpF32Le OpCode = 0x5F
	OpF32Ge OpCode = 0x60

	// Comparisons - f64
	OpF64Eq OpCode = 0x61
	OpF64Ne OpCode = 0x62
	OpF64Lt OpCode = 0x63
	OpF64Gt OpCode = 0x64
	OpF64Le OpCode = 0x65
	OpF64Ge OpCode = 0x66

	// i32 operations
	OpI32Add  OpCode = 0x6A
	OpI32Sub  OpCode = 0x6B
	OpI32Mul  OpCode = 0x6C
	OpI32DivS OpCode = 0x6D
	OpI32DivU OpCode = 0x6E
	OpI32RemS OpCode = 0x6F
	OpI32RemU OpCode = 0x70

	// i64 operations
	OpI64Add  OpCode = 0x7C
	OpI64Sub  OpCode = 0x7D
	OpI64Mul  OpCode = 0x7E
	OpI64DivS OpCode = 0x7F
	OpI64DivU OpCode = 0x80
	OpI64RemS OpCode = 0x81
	OpI64RemU OpCode = 0x82

	// f32 operations
	OpF32Neg OpCode = 0x8C
	OpF32Add OpCode = 0x92
	OpF32Sub OpCode = 0x93
	OpF32Mul OpCode = 0x94
	OpF32Div OpCode = 0x95

	// f64 operations
	OpF64Neg OpCode = 0x9A
	OpF64Add OpCode = 0xA0
	OpF64Sub OpCode = 0xA1
	OpF64Mul OpCode = 0xA2
	OpF64Div OpCode = 0xA3
)
