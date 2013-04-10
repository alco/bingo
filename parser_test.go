package bingo

import (
	"bytes"
	"encoding/binary"
	"testing"
)

var someData = []byte{10, 0, 1, 0, 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'}

func newParser() *Parser {
	return newParserData(someData)
}

func newParserData(data []byte) *Parser {
	return &Parser{bytes.NewReader(data), binary.LittleEndian, 0}
}

func TestEmptyStruct(t *testing.T) {
	empty := struct{}{}
	p := newParser()

	p.EmitReadStruct(&empty)

	if p.offset != 0 {
		t.Error("Non-zero offset after reading into empty struct")
	}
}

func TestEmptySlice(t *testing.T) {
	byteSlice := struct {
		Data []byte
	}{}
	p := newParser()

	p.EmitReadStruct(&byteSlice)

	if p.offset != 0 {
		t.Error("Non-zero offset after reading into zero-length slice (byte)")
	}

	///

	intSlice := struct {
		Data []uint16
	}{}
	p = newParser()

	p.EmitReadStruct(&intSlice)

	if p.offset != 0 {
		t.Error("Non-zero offset after reading into zero-length slice (uint16)")
	}
}

func TestLength32Little(t *testing.T) {
	s := struct {
		Length uint32
	}{}
	p := newParser()
	p.byteOrder = binary.LittleEndian

	p.EmitReadStruct(&s)

	if s.Length != 0x1000A {
		t.Error("Failed to read a single-field struct of uint32 (little):", s.Length)
	}
	if p.offset != 4 {
		t.Error("Invalid offset after reading uint32 (little):", p.offset)
	}
}

func TestLength16Little(t *testing.T) {
	s := struct {
		Length uint16
	}{}
	p := newParser()
	p.byteOrder = binary.LittleEndian

	p.EmitReadStruct(&s)

	if s.Length != 10 {
		t.Error("Failed to read a single-field struct of uint16 (little):", s.Length)
	}
	if p.offset != 2 {
		t.Error("Invalid offset after reading uint16 (little):", p.offset)
	}
}

func TestLength32Big(t *testing.T) {
	s := struct {
		Length uint32
	}{}
	p := newParser()
	p.byteOrder = binary.BigEndian

	p.EmitReadStruct(&s)

	if s.Length != 0x0A000100 {
		t.Error("Failed to read a single-field struct of uint32 (big):", s.Length)
	}
	if p.offset != 4 {
		t.Error("Invalid offset after reading uint32 (big):", p.offset)
	}
}

func TestLength16Big(t *testing.T) {
	s := struct {
		Length uint16
	}{}
	p := newParser()
	p.byteOrder = binary.BigEndian

	p.EmitReadStruct(&s)

	if s.Length != 0x0A00 {
		t.Error("Failed to read a single-field struct of uint16 (big):", s.Length)
	}
	if p.offset != 2 {
		t.Error("Invalid offset after reading uint16 (big):", p.offset)
	}
}

func TestSliceByte(t *testing.T) {
	s := struct {
		Length uint16
		Data   []byte `len:"Length"`
	}{}
	p := newParser()

	p.EmitReadStruct(&s)

	if s.Length != 10 {
		t.Error("Failed to read correct length for slice (uint16):", s.Length)
	}
	if string(s.Data) != "\x01\x00abcdefgh" {
		t.Error("Invalid data read into []byte:", s.Data)
	}
	if p.offset != 12 {
		t.Error("Invalid parser offset after byte slice:", p.offset)
	}
}

func TestSliceInt(t *testing.T) {
	data := []byte{4, 0, 0, 0, 1, 0, 2, 0, 3, 0, 4, 0}
	s := struct {
		Length uint32
		Data   []uint16 `len:"Length"`
	}{}
	p := newParserData(data)

	p.EmitReadStruct(&s)

	if s.Length != 4 {
		t.Error("Failed to read correct length for slice (uint32):", s.Length)
	}
	if uint32(len(s.Data)) != s.Length || !(s.Data[0] == 1 && s.Data[1] == 2 && s.Data[2] == 3 && s.Data[3] == 4) {
		t.Error("Invalid data read into []byte:", s.Data)
	}
	if p.offset != 4+4*2 {
		t.Error("Invalid parser offset after parsing int slice:", p.offset)
	}
}

var fixedSizeData = []byte{'B', 'I', 'N', 'G',
	123, 0,
	3, 2, 1, 111,
	0, 128,
	13, 0, 1, 2,
	0xF2,
	255,
	0, 0, 1, 1, 2, 2, 3, 3}

type FixedSizeStruct struct {
	Signature [4]byte
	Version   uint16
	Reserved  [2]int16
	NChans    int16
	Height    int32
	Width     int8
	Depth     uint8
	ColorMode int64
}

func TestFixedSizeStructLittle(t *testing.T) {
	s := FixedSizeStruct{}
	p := newParserData(fixedSizeData)

	p.EmitReadStruct(&s)

	if string(s.Signature[:]) != "BING" {
		t.Error("Error parsing fixed-size byte array (little):", s.Signature)
	}
	if s.Version != 123 {
		t.Error("Error parsing uint16 (little):", s.Version)
	}
	if !(s.Reserved[0] == 0x203 && s.Reserved[1] == 0x6F01) {
		t.Error("Error parsing [2]int16 (little):", s.Reserved)
	}
	if s.NChans != -32768 {
		t.Error("Error parsing int16 (little):", s.NChans)
	}
	if s.Height != 33619981 {
		t.Error("Error parsing int32 (little):", s.Height)
	}
	if s.Width != -14 {
		t.Error("Error parsing int8 (little):", s.Width)
	}
	if s.Depth != 255 {
		t.Error("Error parsing uint8 (little):", s.Depth)
	}
	if s.ColorMode != 0x0303020201010000 {
		t.Error("Error parsing int64 (little):", s.ColorMode)
	}
	if p.offset != 26 {
		t.Error("Invalid parser offset after fixed-size struct (little):", p.offset)
	}
}

func TestFixedSizeStructBig(t *testing.T) {
	s := FixedSizeStruct{}
	p := newParserData(fixedSizeData)
	p.byteOrder = binary.BigEndian

	p.EmitReadStruct(&s)

	if string(s.Signature[:]) != "BING" {
		t.Error("Error parsing fixed-size byte array (big):", s.Signature)
	}
	if s.Version != 0x7B00 {
		t.Error("Error parsing uint16 (big):", s.Version)
	}
	if !(s.Reserved[0] == 0x302 && s.Reserved[1] == 0x16F) {
		t.Error("Error parsing [2]int16 (big):", s.Reserved)
	}
	if s.NChans != 128 {
		t.Error("Error parsing int16 (big):", s.NChans)
	}
	if s.Height != 0xD000102 {
		t.Error("Error parsing int32 (big):", s.Height)
	}
	if s.Width != -14 {
		t.Error("Error parsing int8 (big):", s.Width)
	}
	if s.Depth != 255 {
		t.Error("Error parsing uint8 (big):", s.Depth)
	}
	if s.ColorMode != 0x0000010102020303 {
		t.Error("Error parsing int64 (big):", s.ColorMode)
	}
	if p.offset != 26 {
		t.Error("Invalid parser offset after fixed-size struct (big):", p.offset)
	}
}
