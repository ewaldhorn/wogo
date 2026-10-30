package compiler

import "bytes"

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
// writeSection writes a wasm binary section in the correct order and format.
func writeSection(buf *bytes.Buffer, sectionID byte, payload []byte) {
	buf.WriteByte(sectionID)              // which section are we writing
	putUvarint(buf, uint32(len(payload))) // how many bytes is it
	buf.Write(payload)                    // the actual section
}

// ------------------------------------------------------------------------------------------------
// writeTypeSection encodes the function signatures (ID 1).
// TODO: See if there's a way in the future to collapse types and not repeat them but reuse, not
// entirely sure if WebAssembly allows that, though I can't see why not.
func writeTypeSection(buf *bytes.Buffer, signatures []funcSignature) {
	if len(signatures) == 0 {
		return
	}

	var payload bytes.Buffer
	putUvarint(&payload, uint32(len(signatures))) // how many function types do we have here

	for _, sig := range signatures {
		payload.WriteByte(0x60) // func type indicator
		putUvarint(&payload, uint32(len(sig.params)))
		payload.Write(sig.params) // params accepts
		putUvarint(&payload, uint32(len(sig.results)))
		payload.Write(sig.results) // value(s) returned
	}

	writeSection(buf, SectionType, payload.Bytes())
}

// ------------------------------------------------------------------------------------------------
// writeFunctionSection maps each function to its signature index (ID 3).
func writeFunctionSection(buf *bytes.Buffer, functions []compiledFunction) {
	if len(functions) == 0 {
		return
	}

	var payload bytes.Buffer
	putUvarint(&payload, uint32(len(functions)))
	for _, fn := range functions {
		putUvarint(&payload, fn.typeIndex)
	}
	writeSection(buf, SectionFunction, payload.Bytes())
}

// ------------------------------------------------------------------------------------------------
// writeExportSection exports functions by name (ID 7).
func writeExportSection(buf *bytes.Buffer, functions []compiledFunction) {
	if len(functions) == 0 {
		return
	}

	const exportKindFunction = 0x00

	var payload bytes.Buffer
	putUvarint(&payload, uint32(len(functions)))
	for i, fn := range functions {
		writeString(&payload, fn.name)
		payload.WriteByte(exportKindFunction)
		putUvarint(&payload, uint32(i))
	}
	writeSection(buf, SectionExport, payload.Bytes())
}

// ------------------------------------------------------------------------------------------------
// writeCodeSection encodes the compiled function bodies (ID 10).
func writeCodeSection(buf *bytes.Buffer, functions []compiledFunction) {
	if len(functions) == 0 {
		return
	}

	var payload bytes.Buffer
	putUvarint(&payload, uint32(len(functions)))
	for _, fn := range functions {
		putUvarint(&payload, uint32(len(fn.body)))
		payload.Write(fn.body)
	}
	writeSection(buf, SectionCode, payload.Bytes())
}
