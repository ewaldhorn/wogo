package compiler

// ------------------------------------------------------------------------------------------------
// Opcodes in WebAssembly.
// See https://webassembly.github.io/spec/core/binary/instructions.html
// And https://webassembly.github.io/spec/core/appendix/index-instructions.html
// TODO: This is going to get messy, I need to find a better way to organise these.
const (
	// Control Flow
	OpEnd      byte = 0x0B
	OpReturn   byte = 0x0F
	OpDrop     byte = 0x1A
	OpLocalGet byte = 0x20

	// Constants
	OpI32Const byte = 0x41
	OpI64Const byte = 0x42
	OpF32Const byte = 0x43
	OpF64Const byte = 0x44

	// Comparisons
	OpI32Eqz byte = 0x45
	OpI64Eqz byte = 0x50

	// i32 operations
	OpI32Add  byte = 0x6A
	OpI32Sub  byte = 0x6B
	OpI32Mul  byte = 0x6C
	OpI32DivS byte = 0x6D
	OpI32DivU byte = 0x6E
	OpI32RemS byte = 0x6F
	OpI32RemU byte = 0x70

	// i64 operations
	OpI64Add  byte = 0x7C
	OpI64Sub  byte = 0x7D
	OpI64Mul  byte = 0x7E
	OpI64DivS byte = 0x7F
	OpI64DivU byte = 0x80
	OpI64RemS byte = 0x81
	OpI64RemU byte = 0x82

	// f32 operations
	OpF32Neg byte = 0x8C
	OpF32Add byte = 0x92
	OpF32Sub byte = 0x93
	OpF32Mul byte = 0x94
	OpF32Div byte = 0x95

	// f64 operations
	OpF64Neg byte = 0x9A
	OpF64Add byte = 0xA0
	OpF64Sub byte = 0xA1
	OpF64Mul byte = 0xA2
	OpF64Div byte = 0xA3
)
