package compiler

import "bytes"

// ================================================================================================
// Little Endian helpers.
//
// WebAssembly binaries are little endian encoded, see https://webassembly.org/docs/portability/
// ================================================================================================

// ------------------------------------------------------------------------------------------------
// writeVarUint encodes an unsigned integer using unsigned LEB128.
func writeVarUint(buf *bytes.Buffer, val uint32) {
	for {
		b := byte(val & 0x7F)
		val >>= 7
		if val != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if val == 0 {
			break
		}
	}
}

// ------------------------------------------------------------------------------------------------
// writeVarInt encodes a signed integer using signed LEB128.
func writeVarInt(buf *bytes.Buffer, val int32) {
	more := true
	for more {
		b := byte(val & 0x7F)
		val >>= 7
		signBit := (b & 0x40) != 0
		if (val == 0 && !signBit) || (val == -1 && signBit) {
			more = false
		} else {
			b |= 0x80
		}
		buf.WriteByte(b)
	}
}
