package ir

// ------------------------------------------------------------------------------------------------
// Param represents a parameter of a wasm function.
type Param struct {
	Name   string
	GoType string
	Type   Type
}

// ------------------------------------------------------------------------------------------------
// Local represents a local variable declared in a wasm function.
type Local struct {
	Name   string
	GoType string
	Type   Type
}

// ------------------------------------------------------------------------------------------------
// Function represents a function definition in the IR module.
type Function struct {
	Name         string
	Params       []Param
	Locals       []Local
	ReturnType   string
	Results      []Type
	Instructions []Instruction
}

// ------------------------------------------------------------------------------------------------
// Module represents a complete WebAssembly module.
type Module struct {
	Functions []Function
}
