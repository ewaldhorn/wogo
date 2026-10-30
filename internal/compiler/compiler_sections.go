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
func writeSection(buf *bytes.Buffer, sectionID byte, payload []byte) {
	buf.WriteByte(sectionID)
	putUvarint(buf, uint32(len(payload)))
	buf.Write(payload)
}
