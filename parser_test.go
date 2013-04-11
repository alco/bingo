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

func isEqualu16(a []uint16, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestEmptyStruct(t *testing.T) {
	empty := struct{}{}
	p := newParser()

	if err := p.EmitReadStruct(&empty); err != nil {
		t.Error(err)
	}

	if p.offset != 0 {
		t.Error("Non-zero offset after reading into empty struct")
	}
}

func TestEmptySlice(t *testing.T) {
	byteSlice := struct {
		Data []byte
	}{}
	p := newParser()

	if err := p.EmitReadStruct(&byteSlice); err != nil {
		t.Error(err)
	}

	if p.offset != 0 {
		t.Error("Non-zero offset after reading into zero-length slice (byte)")
	}

	///

	intSlice := struct {
		Data []uint16
	}{}
	p = newParser()

	if err := p.EmitReadStruct(&intSlice); err != nil {
		t.Error(err)
	}

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

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if s.Length != 4 {
		t.Error("Failed to read correct length for slice (uint32):", s.Length)
	}
	if !(uint32(len(s.Data)) == s.Length && isEqualu16(s.Data, []uint16{1,2,3,4})) {
		t.Error("Invalid data read into []byte:", s.Data)
	}
	if p.offset != uint32(len(data)) {
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
	if s.Signature[0] != 'B' {
		return errors.New("verification failure")
	}

	return nil
}

func TestFixedSizeStructLittle(t *testing.T) {
	s := FixedSizeStruct{}
	p := newParserData(fixedSizeData)

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

type UnicodeString struct {
	Length uint32
	Chars  []uint16 `len:"Length"`
}

func TestCustomType(t *testing.T) {
	data := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		4, 0, 0, 0,
		'a', 0, 'b', 0, 'c', 0, 'd', 0}
	s := struct {
		SomeData [10]byte
		Name     UnicodeString
	}{}

	p := newParserData(data)

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

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

func TestCustomSliceZero(t *testing.T) {
	data := []byte{0, 0, 0, 0, 0}
	type VarStruct struct {
		DataLength uint16
		Data       []byte `len:"DataLength"`
	}
	s := struct {
		SomeData uint32
		Count    int8
		Slice    []VarStruct `len:"Count"`
	}{}
	p := newParserData(data)

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if s.SomeData != 0 {
		t.Error("Error parsing uint32:", s.SomeData)
	}
	if s.Count != 0 {
		t.Error("Error parsing int8:", s.Count)
	}
	if len(s.Slice) != 0 {
		t.Error("Error parsing zero-length varsize slice:", s.Slice)
	}
	if p.offset != 5 {
		t.Error("Invalid offset after zero-length varsize slice:", p.offset)
	}
}

func TestCustomSlice(t *testing.T) {
	data := []byte{0, 0, 0, 0, // SomeData
		3,    // Count
		0, 0, // DataLength
		4, 0, // DataLength
		'a', 'b', 'c', 'd', // Data
		1, 0, // DataLength
		13} // Data
	type VarStruct struct {
		DataLength uint16
		Data       []byte `len:"DataLength"`
	}
	s := struct {
		SomeData uint32
		Count    int8
		Slice    []VarStruct `len:"Count"`
	}{}
	p := newParserData(data)

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if s.SomeData != 0 {
		t.Error("Error parsing uint32:", s.SomeData)
	}
	if s.Count != 3 {
		t.Error("Error parsing int8:", s.Count)
	}
	if len(s.Slice) != int(s.Count) {
		t.Error("Error parsing varsize slice (len):", s.Slice)
	}
	if s.Slice[0].DataLength != 0 {
		t.Error("Error parsing varsize slice (elem 0):", s.Slice)
	}
	if string(s.Slice[1].Data) != "abcd" {
		t.Error("Error parsing varsize slice (elem 1):", s.Slice)
	}
	if !(s.Slice[2].DataLength == 1 && s.Slice[2].Data[0] == 13) {
		t.Error("Error parsing varsize slice (elem 2):", s.Slice)
	}
	if p.offset != 16 {
		t.Error("Invalid offset after varsize slice:", p.offset)
	}
}

type DescriptorT struct {
	ClassIDString UnicodeString
	ClassID       [4]byte `if:"ShouldParseClassID"`
}

func (d *DescriptorT) ShouldParseClassID() bool {
	return d.ClassIDString.Length == 0
}

func TestOptionalField(t *testing.T) {
	data := []byte{1, 0, 0, 0,
		'a', 0,
		'a', 'b', 'c', 'd'}
	s := DescriptorT{}
	p := newParserData(data)

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if !(s.ClassIDString.Length == 1 && s.ClassIDString.Chars[0] == 'a') {
		t.Error("Error parsing UnicodeString (optional):", s.ClassIDString)
	}
	if !(s.ClassID[0] == 0 && s.ClassID[1] == 0 && s.ClassID[2] == 0 && s.ClassID[3] == 0) {
		t.Error("Read too much data after UnicodeString (optional):", s.ClassID)
	}
	if p.offset != 6 {
		t.Error("Invalid offset after UnicodeString (optional):", p.offset)
	}
}

func TestOptionalFieldSecondChoice(t *testing.T) {
	data := []byte{0, 0, 0, 0,
		'a', 'b', 'c', 'd'}
	s := DescriptorT{}
	p := newParserData(data)

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if !(s.ClassIDString.Length == 0 && len(s.ClassIDString.Chars) == 0) {
		t.Error("Error parsing UnicodeString (second optional):", s.ClassIDString)
	}
	if !(s.ClassID[0] == 'a' && s.ClassID[1] == 'b' && s.ClassID[2] == 'c' && s.ClassID[3] == 'd') {
		t.Error("Error parsing optional ClassID:", s.ClassID)
	}
	if p.offset != 8 {
		t.Error("Invalid offset after second optional:", p.offset)
	}
}

type SlicesHeader struct {
	Top, Left, Bottom, Right byte
	GroupName                UnicodeString
	Count                    uint32  // number of slices to follow
	Slices                   []Block `len:"Count"`

	SlicesHeaderExtra
}

type Block struct {
	Name, URL, Meta         UnicodeString
	Alpha, Red, Green, Blue byte

	DescriptorT
}

type SlicesHeaderExtra struct {
	DescriptorVersion uint32
	Descriptor        DescriptorT
}

/*
type DescriptorT struct {
	ClassIDString UnicodeString
	ClassID       [4]byte `if:"ShouldParseClassID"`
}
*/

func TestDoubleNestedStruct(t *testing.T) {
	data := []byte{1, 2, 3, 4, // top, left, bottom, right
		0, 0, 0, 0, // UnicodeString
		2, 0, 0, 0, // Count

		3, 0, 0, 0, // UnicodeString.Length
		'a', 0, 1, 1, 'c', 0,
		2, 0, 0, 0, // UnicodeString.Length
		'a', 0, 'b', 0,
		1, 0, 0, 0, // UnicodeString.Length
		13, 12,
		4, 3, 2, 1, // RGBA
		1, 0, 0, 0,
		'a', 0,

		0, 0, 0, 0, // UnicodeString.Length
		0, 0, 0, 0, // UnicodeString.Length
		0, 0, 0, 0, // UnicodeString.Length
		4, 3, 2, 1, // RGBA
		0, 0, 0, 0,
		'A', 'B', 'C', 'D',

		1, 2, 3, 4, // DescriptorVersion
		2, 0, 0, 0,
		'a', 'b', 'c', 'd'}
	s := SlicesHeader{}
	p := newParserData(data)

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if !(s.Top == 1 && s.Left == 2 && s.Bottom == 3 && s.Right == 4) {
		t.Error("Error parsing first row of bytes (nested):", s)
	}
	if !(s.GroupName.Length == 0 && len(s.GroupName.Chars) == 0) {
		t.Error("Error parsing GroupName (nested):", s.GroupName)
	}
	if s.Count != 2 {
		t.Error("Error parsing Count (nested):", s.Count)
	}
	if !(s.Slices[0].Name.Length == 3 && len(s.Slices[0].Name.Chars) == 3) {
		t.Error("Error parsing first block's Name (nested):", s.Slices[0])
	}
	if !(s.Slices[0].URL.Length == 2 && len(s.Slices[0].URL.Chars) == 2) {
		t.Error("Error parsing first block's URL (nested):", s.Slices[0])
	}
	if !(s.Slices[0].Meta.Length == 1 && len(s.Slices[0].Meta.Chars) == 1) {
		t.Error("Error parsing first block's Meta (nested):", s.Slices[0])
	}
	if !(len(s.Slices[0].ClassIDString.Chars) == 1 && s.Slices[0].ClassIDString.Chars[0] == 'a') {
		t.Error("Error parsing first block's ClassIDString (nested):", s.Slices[0])
	}
	// FIXME: unfinished checks
	if p.offset != 82 {
		t.Error("Invalid offset after double nested struct:", p.offset)
	}
}

func TestUnkownLengthSlice(t *testing.T) {
	data := []byte{10,
				   0, 0, 0, 0,
				   1, 0, 0, 0,
				   'a', 0}
	s := struct {
		Size int8
		Elems []UnicodeString `size:"Size"`
	}{}
	p := newParserData(data)

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if s.Size != 10 {
		t.Error("Error parsing int8:", s.Size)
	}
	if len(s.Elems) != 2 {
		t.Error("Error determining correct number of elements to parse:", s.Elems)
	}
	if s.Elems[0].Length != 0 {
		t.Error("Error parsing the first element: empty UnicodeString:", s.Elems[0])
	}
	if !(s.Elems[1].Length == 1 && len(s.Elems[1].Chars) == 1 && s.Elems[1].Chars[0] == 'a') {
		t.Error("Error parsing second elements -- 'a' UnicodeString:", s.Elems[1])
	}
	if p.offset != 11 {
		t.Error("Invalid offset after unknown length slice:", p.offset)
	}
}

/* Next up */

// Challenges:
// * bool, string
// * read until EOF
// * read UNKNOWN elements into slice until read N bytes
// * slice of pointers
