package compiler

import "bytes"

// ------------------------------------------------------------------------------------------------
func writeSection(buf *bytes.Buffer, sectionID byte, payload []byte) {
	buf.WriteByte(sectionID)
	putUvarint(buf, uint32(len(payload)))
	buf.Write(payload)
}
