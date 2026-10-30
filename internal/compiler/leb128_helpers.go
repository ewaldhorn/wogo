package compiler

import (
	"bytes"
)

// ================================================================================================
// Little Endian helpers.
//
// WebAssembly binaries are little endian encoded, see https://webassembly.org/docs/portability/
// ================================================================================================

// ------------------------------------------------------------------------------------------------
// putUvarint encodes an unsigned integer using unsigned LEB128.
func putUvarint(buf *bytes.Buffer, val uint32) {
	for val >= 0x80 {
		buf.WriteByte(byte(val) | 0x80)
		val >>= 7
	}
	buf.WriteByte(byte(val))
}

// ------------------------------------------------------------------------------------------------
// putVarint encodes a signed integer using signed LEB128.
func putVarint(buf *bytes.Buffer, val int32) {
	for {
		b := byte(val & 0x7F)
		val >>= 7
		signBit := (b & 0x40) != 0
		if (val == 0 && !signBit) || (val == -1 && signBit) {
			buf.WriteByte(b)
			return
		}
		buf.WriteByte(b | 0x80)
	}
}
