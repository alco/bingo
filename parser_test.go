package bingo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
	"unicode/utf16"
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

func (s *FixedSizeStruct) Verify() error {
	if s.Signature[0] != '' {
		return errors.New("verification failure")
	}

	return nil
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

func TestCustomType(t *testing.T) {
	data := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		4, 0, 0, 0,
		'a', 0, 'b', 0, 'c', 0, 'd', 0}
	type UnicodeString struct {
		Length uint32
		Chars  []uint16 `len:"Length"`
	}
	s := struct {
		SomeData [10]byte
		Name     UnicodeString
	}{}

	p := newParserData(data)

	p.EmitReadStruct(&s)

	if s.Name.Length != 4 {
		t.Error("Error parsing nested fixed uint32:", s.Name.Length)
	}
	if string(utf16.Decode(s.Name.Chars)) != "abcd" {
		t.Error("Error parsing nested []uint16:", s.Name.Chars)
	}
	if p.offset != 10+4+4*2 {
		t.Error("Invalid offset after custom type UnicodeString:", p.offset)
	}
}

func TestCustomTypeEmbed(t *testing.T) {
	data := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		4,
		'a', 'b', 'c', 'd'}
	type pascalStringEmbed struct {
		Length uint8
		Chars  []byte `len:"Length"`
	}
	type PascalString struct {
		pascalStringEmbed
	}
	s := struct {
		SomeData [10]byte
		Name     PascalString
	}{}

	p := newParserData(data)

	p.EmitReadStruct(&s)

	if s.Name.Length != 4 {
		t.Error("Error parsing nested fixed uint8:", s.Name.Length)
	}
	if string(s.Name.Chars) != "abcd" {
		t.Error("Error parsing nested []byte:", s.Name.Chars)
	}
	if p.offset != 10+1+4 {
		t.Error("Invalid offset after custom type PascalString:", p.offset)
	}
}

func TestCustomTypePad(t *testing.T) {
	data := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		4,
		'a', 'b', 'c', 'd', 0}
	type pascalStringEmbed struct {
		Length uint8
		Chars  []byte `len:"Length"`
	}
	type PascalString struct {
		pascalStringEmbed `pad:"2"`
	}
	s := struct {
		SomeData [10]byte
		Name     PascalString
	}{}

	p := newParserData(data)

	p.EmitReadStruct(&s)

	if s.Name.Length != 4 {
		t.Error("Error parsing nested fixed uint8:", s.Name.Length)
	}
	if string(s.Name.Chars) != "abcd" {
		t.Error("Error parsing nested []byte:", s.Name.Chars)
	}
	if p.offset != 10+1+4+1 { // +1 byte for padding
		t.Error("Invalid offset after custom padded type PascalString:", p.offset)
	}
}

func TestCustomTypeZeroPad(t *testing.T) {
	data := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0}
	type pascalStringEmbed struct {
		Length uint8
		Chars  []byte `len:"Length"`
	}
	type PascalString struct {
		pascalStringEmbed `pad:"2"`
	}
	s := struct {
		SomeData [10]byte
		Name     PascalString
	}{}

	p := newParserData(data)

	p.EmitReadStruct(&s)

	if s.Name.Length != 0 {
		t.Error("Error parsing nested fixed uint8:", s.Name.Length)
	}
	if len(s.Name.Chars) != 0 {
		t.Error("Error parsing nested []byte:", s.Name.Chars)
	}
	if p.offset != 10+2 {
		t.Error("Invalid offset after custom padded type PascalString with length 0:", p.offset)
	}
}

func TestPaddedSlice(t *testing.T) {
	data := []byte{4, 'a', 'b', 'c', 'd', 0, 0}
	s := struct {
		DataLength uint8
		Data       []byte `len:"DataLength" pad:"3"`
	}{}
	p := newParserData(data)

	p.EmitReadStruct(&s)
	if s.DataLength != 4 {
		t.Error("Error parsing uint8 length:", s.DataLength)
	}
	if string(s.Data) != "abcd" {
		t.Error("Error parsing []byte:", s.Data)
	}
	if p.offset != 1+4+2 { // +2 bytes for padding
		t.Error("Invalid offset after padded []byte:", p.offset)
	}
}

func TestPaddedSliceZero(t *testing.T) {
	data := []byte{0, 0}
	s := struct {
		DataLength uint16
		Data       []byte `len:"DataLength" pad:"2"`
	}{}
	p := newParserData(data)

	p.EmitReadStruct(&s)
	if s.DataLength != 0 {
		t.Error("Error parsing uint16 length:", s.DataLength)
	}
	if len(s.Data) != 0 {
		t.Error("Error parsing []byte:", s.Data)
	}
	if p.offset != 2 {
		t.Error("Invalid offset after zero-length []byte:", p.offset)
	}
}


/* Next up */

// Challenges:
// * nested slice of structs (correct offset)
// * nested anonymous varsize struct
// * bool, string
// * optional field (ClassIDString && ClassID)
type SlicesHeader struct {
	Version                  uint32 // == 6, 7, or 8
	Top, Left, Bottom, Right uint32 // bounding rectangle for all of the slices
	GroupName                string // name of group of slices: Unicode string
	Count                    uint32 // number of slices to follow
	Slices                   []SlicesResourceBlock

	SlicesHeaderCS
}

type SlicesHeaderCS struct {
	DescriptorVersion uint32 // == 16 for Photoshop 6.0
	Descriptor        DescriptorT
}

type SlicesResourceBlock struct {
	ID, GroupID, Origin          uint32
	LayerID                      uint32 // associated Layer ID (only present if Origin == 1)
	Name                         string // Unicode string
	Type                         uint32
	Left, Top, Right, Bottom     uint32
	URL, Target, Message, AltTag string // Unicode string
	isHTML                       bool   // cell text is HTML
	CellText                     string // Unicode string
	AlignHoriz                   uint32 // horizontal alignment
	AlignVert                    uint32 // vertical alignment
	Alpha, Red, Green, Blue      bool

	// for Photoshop > 6.0
	DescriptorVersion uint32
	Descriptor        DescriptorT
}

type DescriptorT struct {
	Name          string  // Unicode string: name from classID
	ClassIDString string  // "" if ClassIDLength == 0
	ClassID       [4]byte // classID if ClassIDString  == ""
	NItems        uint32
}
