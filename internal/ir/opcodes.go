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

	// Comparisons
	OpI32Eqz OpCode = 0x45
	OpI64Eqz OpCode = 0x50

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
