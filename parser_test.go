package bingo

import (
	"bytes"
	"errors"
	"testing"
	"unicode/utf16"
)

var someData = []byte{10, 0, 1, 0, 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'}

func newParser() *Parser {
	return newParserData(someData)
}

func newParserData(data []byte) *Parser {
	return NewParser(bytes.NewReader(data), LittleEndian, Default)
}


func TestNonStruct(t *testing.T) {
	thing := 15
	p := newParser()

	if err := p.EmitReadStruct(&thing); err != nil {
		if perr, ok := err.(*ParseError); !ok || perr.Error() != "Invalid argument type *int. Expected pointer to a struct." {
			t.Error("Incorrect error:", err)
		}
	} else {
		t.Error()
	}
}

func TestNonPointer(t *testing.T) {
	s := struct{}{}
	p := newParser()

	if err := p.EmitReadStruct(s); err != nil {
		if perr, ok := err.(*ParseError); !ok || perr.Error() != "Invalid argument type struct {}. Expected pointer to a struct." {
			t.Error("Incorrect error:", err)
		}
	} else {
		t.Error()
	}
}

func TestEmptyStruct(t *testing.T) {
	empty := struct{}{}
	p := newParser()

	if err := p.EmitReadStruct(&empty); err != nil {
		t.Error(err)
	}

	if p.offset != 0 {
		t.Error("Non-zero offset:", p.offset)
	}
}

func TestPtrField(t *testing.T) {
	s := struct {
		Data *int8
	}{}
	p := newParser()

	if err := p.EmitReadStruct(&s); err != nil {
		if perr, ok := err.(*ParseError); !ok || perr.Error() != "Error reading field 'Data *int8'. Pointer fields are not supported." {
			t.Error("Incorrect error:", err)
		}
	} else {
		t.Error()
	}

	if p.offset != 0 {
		t.Error("Invalid offset:", p.offset)
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
	p.byteOrder = LittleEndian

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
	p.byteOrder = LittleEndian

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
	p.byteOrder = BigEndian

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
	p.byteOrder = BigEndian

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

func TestBoolField(t *testing.T) {
	s := struct {
		F bool
	}{}
	p := newParser()

	if err := p.EmitReadStruct(&s); err != nil {
		if perr, ok := err.(*ParseError); !ok || perr.Error() != "Error reading field 'F bool'. Type not supported." {
			t.Error("Incorrect error:", err)
		}
	} else {
		t.Error()
	}

	if p.offset != 0 {
		t.Error("Invalid offset:", p.offset)
	}
}

func TestEmptyLenTag(t *testing.T) {
	s := struct {
		Data   []byte `len:""`
	}{}
	p := newParser()

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if p.offset != 0 {
		t.Error("Invalid offset:", p.offset)
	}
}

func TestInvalidLenTag(t *testing.T) {
	s := struct {
		Data   []byte `len:"Lengthy"`
	}{}
	p := newParser()

	if err := p.EmitReadStruct(&s); err != nil {
		if perr, ok := err.(*ParseError); !ok || perr.Error() != "Field 'Lengthy' for 'Data []uint8' not found. Referenced from a `len` tag." {
			t.Error("Incorrect error:", err)
		}
	} else {
		t.Error()
	}

	if p.offset != 0 {
		t.Error("Invalid offset:", p.offset)
	}
}

type InvalidLenStruct struct {
	Length int8
	Data   []byte `len:"Length()"`
}

func TestInvalidLenFuncTag(t *testing.T) {
	s := InvalidLenStruct{}
	p := newParser()

	if err := p.EmitReadStruct(&s); err != nil {
		if perr, ok := err.(*ParseError); !ok || perr.Error() != "Method 'Length()' for '*bingo.InvalidLenStruct' not found. Referenced from a `len` tag." {
			t.Error("Incorrect error:", err)
		}
	} else {
		t.Error()
	}

	if p.offset != 1 {
		t.Error("Invalid offset:", p.offset)
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
	if !(uint32(len(s.Data)) == s.Length && isEqualu16(s.Data, []uint16{1, 2, 3, 4})) {
		t.Error("Invalid data read into []byte:", s.Data)
	}
	if p.offset != uint(len(data)) {
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

func (s *FixedSizeStruct) Verify(p *Parser) error {
	if s.Signature[0] != 'B' {
		return errors.New("verification failure")
	}

	ctx := p.Context().(*FixedSizeStruct)
	ctx.Version++

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
	if s.Version != 124 {
		t.Error("Error verifying the version (little):", s.Version)
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
	p.byteOrder = BigEndian

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if string(s.Signature[:]) != "BING" {
		t.Error("Error parsing fixed-size byte array (big):", s.Signature)
	}
	if s.Version != 0x7B01 {
		t.Error("Error verifying the version (big):", s.Version)
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

func TestInvalidPadding(t *testing.T) {
	s := struct {
		Data []byte `pad:"string"`
	}{}
	p := newParser()

	if err := p.EmitReadStruct(&s); err != nil {
		if perr, ok := err.(*ParseError); !ok || perr.Error() != "Invalid value for `pad` tag: string. Expected an integer." {
			t.Error("Incorrect error:", err)
		}
	} else {
		t.Error()
	}

	if p.offset != 0 {
		t.Error("Invalid offset:", p.offset)
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

func (d *DescriptorT) ShouldParseClassID(p *Parser) bool {
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

type Lengthy struct {
	ShortLength int8  `if:"UseShortLength"`
	LongLength  int32 `if:"!UseShortLength"`
	pred        func() bool
}

func (l *Lengthy) UseShortLength(p *Parser) bool {
	return l.pred()
}

func TestOptionalFieldWithNegation(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	s := Lengthy{}
	s.pred = func() bool { return false }
	p := newParserData(data)

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if !(s.ShortLength == 0 && s.LongLength == 0x04030201) {
		t.Error("Error using negative condition:", s)
	}
	if p.offset != 4 {
		t.Error("Invalid offset after negative condition:", p.offset)
	}
}

func TestOptionalFieldWithNegation2(t *testing.T) {
	data := []byte{17}
	s := Lengthy{}
	s.pred = func() bool { return true }
	p := newParserData(data)

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if !(s.ShortLength == 17 && s.LongLength == 0) {
		t.Error("Error using negative condition 2:", s)
	}
	if p.offset != 1 {
		t.Error("Invalid offset after negative condition 2:", p.offset)
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
		Size  int8
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

func TestConflictingTags(t *testing.T) {
	s := struct {
		Data []byte `len:"Length" size:"Size"`
	}{}
	p := newParser()

	if err := p.EmitReadStruct(&s); err != nil {
		if perr, ok := err.(*ParseError); !ok || perr.Error() != "Error parsing field 'Data []uint8'. Can't have both `len` and `size` tags on the same field." {
			t.Error("Incorrect error:", err)
		}
	} else {
		t.Error()
	}
	if p.offset != 0 {
		t.Error("Invalid offset:", p.offset)
	}
}

func TestReadUntilEOF(t *testing.T) {
	data := []byte("Hello world!")
	s := struct {
		Data []byte `len:"<inf>"`
	}{}
	p := newParserData(data)

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if string(s.Data) != "Hello world!" {
		t.Error("Error reading until EOF into []byte:", s.Data)
	}
	if p.offset != uint(len(data)) {
		t.Error("Invalid offset after reading until EOF into []byte:", p.offset)
	}
}

type WrongVerifier struct {
}

func (w *WrongVerifier) Verify() {
}

func TestVerifyDetect(t *testing.T) {
	s := WrongVerifier{}
	p := newParser()

	if err := p.EmitReadStruct(&s); err != nil {
		if perr, ok := err.(*ParseError); !ok || perr.Error() != "Type *bingo.WrongVerifier has a Verify() method with incorrect signature. Expected: Verify(p *bingo.Parser) error." {
			t.Error("Incorrect error:", err)
		}
	} else {
		t.Error()
	}

	if p.offset != 0 {
		t.Error("Invalid offset:", p.offset)
	}
}

type Unexported struct {
	dummy FixedSizeStruct
}

func TestNonStrictMode(t *testing.T) {
	s := Unexported{}
	p := newParser()

	if err := p.EmitReadStruct(&s); err != nil {
		t.Error(err)
	}

	if p.offset != 0 {
		t.Error("Invalid offset:", p.offset)
	}
}

func TestStrictMode(t *testing.T) {
	s := Unexported{}
	p := NewParser(nil, BigEndian, Strict)

	if err := p.EmitReadStruct(&s); err != nil {
		if perr, ok := err.(*ParseError); !ok || perr.Error() != "Unable to parse into 'dummy bingo.FixedSizeStruct'. Unexported fields are not supported." {
			t.Error("Incorrect error:", err)
		}
	} else {
		t.Error()
	}

	if p.offset != 0 {
		t.Error("Invalid offset:", p.offset)
	}
}

func TestPanickyMode(t *testing.T) {
	defer func() {
	}()
}

/* Next up */

// Challenges:
// * bool, string
// * slice of pointers
// * precise error reporting


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
