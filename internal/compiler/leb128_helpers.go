package compiler

import (
	"bytes"
	"encoding/binary"
	"math"
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
func putVarint(buf *bytes.Buffer, val int64) {
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

// ------------------------------------------------------------------------------------------------
// putFloat32 encodes a 32-bit floating point number in IEEE 754 little-endian format.
func putFloat32(buf *bytes.Buffer, val float32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], math.Float32bits(val))
	buf.Write(b[:])
}

// ------------------------------------------------------------------------------------------------
// putFloat64 encodes a 64-bit floating point number in IEEE 754 little-endian format.
func putFloat64(buf *bytes.Buffer, val float64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], math.Float64bits(val))
	buf.Write(b[:])
}
