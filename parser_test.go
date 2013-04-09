package goblin

import (
    "testing"
    "bytes"
    "encoding/binary"
)

var someData = []byte{10, 0, 1, 0}

func newParser() *Parser {
    return &Parser{bytes.NewReader(someData), binary.LittleEndian, 0}
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
