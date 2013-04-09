package goblin

import "io"

type Parser struct {
    r io.Reader
    offset uint32
}

func (p *Parser) EmitReadStruct(data interface{}) {

}
