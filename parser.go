package goblin

import (
    "encoding/binary"
    "io"
)

type Parser struct {
    r io.Reader
    byteOrder binary.ByteOrder
    offset uint32
}

func (p *Parser) EmitReadStruct(data interface{}) {
    // Try fast path for fixed-size data
    if p.EmitReadFixed(data) {
        return
    }

    // TODO
}

func (p *Parser) EmitReadFixed(data interface{}) bool {
	size := binary.Size(data)
    if size < 0 {
        return false
    }

	err := binary.Read(p.r, p.byteOrder, data)
	if err != nil {
        p.RaiseError(err)
	}
	p.offset += uint32(size)

    return true
}

func (p *Parser) RaiseError(err error) {
    panic(err)
}
