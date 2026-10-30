package compiler

import "io"

// ------------------------------------------------------------------------------------------------
// WebAssembly binary magic and version numbers.
// These seem to be static across all WASM files I've examined.
// See https://webassembly.github.io/spec/core/binary/modules.html
var (
	wasmMagic   = []byte{0x00, 0x61, 0x73, 0x6d} // \0asm
	wasmVersion = []byte{0x01, 0x00, 0x00, 0x00} // version 1
)

// ------------------------------------------------------------------------------------------------
// WebAssembly binary specification gives these section IDs.
// See https://webassembly.github.io/spec/core/binary/modules.html
const (
	SectionType      byte = 1
	SectionImport    byte = 2
	SectionFunction  byte = 3
	SectionTable     byte = 4
	SectionMemory    byte = 5
	SectionGlobal    byte = 6
	SectionExport    byte = 7
	SectionStart     byte = 8
	SectionElement   byte = 9
	SectionCode      byte = 10
	SectionData      byte = 11
	SectionDataCount byte = 12
	SectionTag       byte = 13
)

// ------------------------------------------------------------------------------------------------
// Writes the standard wasm binary header (magic number and version)
// Does basic error handling by returning an error or not.
func writeStandardWasmBinaryHeader(w io.Writer) error {
	if _, err := w.Write(wasmMagic); err != nil {
		return err
	}

	_, err := w.Write(wasmVersion)
	return err
}
