package goblin

import (
    "testing"
    "bytes"
)

var someData = []byte{10, 0, 0, 0}

func TestEmptyStruct(t *testing.T) {
    empty := struct {}{}
    p := Parser{bytes.NewReader(someData), 0}
    p.EmitReadStruct(&empty)
}
