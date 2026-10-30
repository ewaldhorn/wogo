package compiler

// ------------------------------------------------------------------------------------------------
// Number types in WebAssembly.
// See https://webassembly.github.io/spec/core/binary/types.html
const (
	TypeI32 byte = 0x7F
	TypeI64 byte = 0x7E
	TypeF32 byte = 0x7D
	TypeF64 byte = 0x7C
)

// ------------------------------------------------------------------------------------------------
// Opcodes in WebAssembly.
// See https://webassembly.github.io/spec/core/binary/instructions.html
// And https://webassembly.github.io/spec/core/appendix/index-instructions.html
// TODO: This is going to get messy, I need to find a better way to organise these.
const (
	OpReturn   byte = 0x0F
	OpLocalGet byte = 0x20
	OpI32Const byte = 0x41
	OpI32Add   byte = 0x6A
	OpI32Sub   byte = 0x6B
	OpI32Mul   byte = 0x6C
	OpI32DivS  byte = 0x6D
	OpI32RemS  byte = 0x6F
	OpEnd      byte = 0x0B
)
