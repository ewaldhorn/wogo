package compiler

import (
	"bytes"
	"testing"
)

// ------------------------------------------------------------------------------------------------
func Test_putUvarint(t *testing.T) {
	tests := []struct {
		val  uint32
		want []byte
	}{
		{0, []byte{0x00}},
		{1, []byte{0x01}},
		{127, []byte{0x7F}},
		{128, []byte{0x80, 0x01}},
		{255, []byte{0xFF, 0x01}},
		{300, []byte{0xAC, 0x02}},
		{624485, []byte{0xE5, 0x8E, 0x26}},
		{0xFFFFFFFF, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x0F}},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		putUvarint(&buf, tt.val)
		if !bytes.Equal(buf.Bytes(), tt.want) {
			t.Errorf("putUvarint(%d) = %X, want %X", tt.val, buf.Bytes(), tt.want)
		}
	}
}

// ------------------------------------------------------------------------------------------------
func Test_putVarint(t *testing.T) {
	tests := []struct {
		val  int64
		want []byte
	}{
		{0, []byte{0x00}},
		{1, []byte{0x01}},
		{-1, []byte{0x7F}},
		{63, []byte{0x3F}},
		{-64, []byte{0x40}},
		{64, []byte{0xC0, 0x00}},
		{-65, []byte{0xBF, 0x7F}},
		{127, []byte{0xFF, 0x00}},
		{-128, []byte{0x80, 0x7F}},
		{128, []byte{0x80, 0x01}},
		{-123456, []byte{0xC0, 0xBB, 0x78}},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		putVarint(&buf, tt.val)
		if !bytes.Equal(buf.Bytes(), tt.want) {
			t.Errorf("putVarint(%d) = %X, want %X", tt.val, buf.Bytes(), tt.want)
		}
	}
}
