package compiler

import "bytes"

// ------------------------------------------------------------------------------------------------
func writeString(buf *bytes.Buffer, s string) {
	putUvarint(buf, uint32(len(s)))
	buf.WriteString(s)
}
