package ir

import (
	"bytes"
	"encoding/binary"
	"math"
)

// ------------------------------------------------------------------------------------------------
// PutUvarint encodes a uint32 in unsigned LEB128 format.
func PutUvarint(buf *bytes.Buffer, val uint32) {
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
// PutVarint encodes an int64 in signed LEB128 format.
func PutVarint(buf *bytes.Buffer, val int64) {
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

// ------------------------------------------------------------------------------------------------
// PutFloat32 encodes a 32-bit floating point number in IEEE 754 little-endian format.
func PutFloat32(buf *bytes.Buffer, val float32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], math.Float32bits(val))
	buf.Write(b[:])
}

// ------------------------------------------------------------------------------------------------
// PutFloat64 encodes a 64-bit floating point number in IEEE 754 little-endian format.
func PutFloat64(buf *bytes.Buffer, val float64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], math.Float64bits(val))
	buf.Write(b[:])
}

// ------------------------------------------------------------------------------------------------
// EncodeInstruction writes the binary WebAssembly encoding of an IR instruction.
func EncodeInstruction(buf *bytes.Buffer, inst Instruction) {
	buf.WriteByte(byte(inst.Op))
	switch inst.Op {
	case OpLocalGet:
		PutUvarint(buf, inst.LocalIdx)
	case OpI32Const:
		PutVarint(buf, int64(inst.I32))
	case OpI64Const:
		PutVarint(buf, inst.I64)
	case OpF32Const:
		PutFloat32(buf, inst.F32)
	case OpF64Const:
		PutFloat64(buf, inst.F64)
	}
}
