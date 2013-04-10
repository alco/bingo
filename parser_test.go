package bingo

import (
    "testing"
    "bytes"
    "encoding/binary"
)

var someData = []byte{10, 0, 1, 0, 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'}

func newParser() *Parser {
    return newParserData(someData)
}

func newParserData(data []byte) *Parser {
    return &Parser{bytes.NewReader(data), binary.LittleEndian, 0}
}

func TestEmptyStruct(t *testing.T) {
    empty := struct {}{}
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
        Data []byte `len:"Length"`
    }{}
    p := newParser()

    p.EmitReadStruct(&s)

    if s.Length != 10 {
        t.Error("Failed to read correct length for slice (uint16):", s.Length)
    }

    if string(s.Data) != "\x01\x00abcdefgh" {
        t.Error("Invalid data read into []byte:", s.Data)
    }
}

func TestSliceInt(t *testing.T) {
    data := []byte{4,0,0,0,1,0,2,0,3,0,4,0}
    s := struct {
        Length uint32
        Data []uint16 `len:"Length"`
    }{}
    p := newParserData(data)

    p.EmitReadStruct(&s)

    if s.Length != 4 {
        t.Error("Failed to read correct length for slice (uint32):", s.Length)
    }

    if uint32(len(s.Data)) != s.Length || !(s.Data[0] == 1 && s.Data[1] == 2 && s.Data[2] == 3 && s.Data[3] == 4) {
        t.Error("Invalid data read into []byte:", s.Data)
    }
}
