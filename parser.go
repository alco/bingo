package bingo

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
	/*"fmt"*/
)

type Parser struct {
	r         io.Reader
	byteOrder binary.ByteOrder
	offset    uint32
}

func (p *Parser) EmitReadStruct(data interface{}) {
	// Try fast path for fixed-size data
	if p.EmitReadFixed(data) {
		return
	}

	// Start reflecting
	ptrtyp := reflect.TypeOf(data)
	typ := ptrtyp.Elem()
	if typ.Kind() != reflect.Struct {
		p.RaiseError(errors.New("Expected a pointer to a struct"))
	}
	ptrval := reflect.ValueOf(data)
	val := ptrval.Elem()

	i := 0
	nfields := typ.NumField()
	for i < nfields {
		pendingBytes := 0
		j := i
		for ; i < nfields; i++ {
			iface := val.Field(i).Interface()
			fieldSize := binary.Size(iface)
			if fieldSize <= 0 {
				break
			}
			pendingBytes += fieldSize
		}

		// We can now read `pendingBytes` bytes before proceeding
		if pendingBytes > 0 {
			buf := p.EmitReadNBytes(pendingBytes)
			d := &decoder{order: p.byteOrder, buf: buf, firstField: j, lastField: i}
			d.value(val)
		}

		for ; i < nfields; i++ {
			fieldval := val.Field(i)
			fieldtyp := typ.Field(i)
			switch fieldval.Kind() {
			case reflect.Slice:
				lenkey := fieldtyp.Tag.Get("len")
				var lenfield reflect.Value
				if len(lenkey) > 0 {
					lenfield = val.FieldByName(lenkey)

					var length int
					switch lenfield.Kind() {
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						length = int(lenfield.Int())
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
						length = int(lenfield.Uint())
					default:
						p.RaiseError(errors.New("Unsupported type for length spec. Only integers are supported."))
					}

					slice := reflect.MakeSlice(fieldtyp.Type, length, length)
					islice := slice.Interface()
					/*p.EmitReadFixedFast(islice, length * int(fieldtyp.Type.Size()))*/
					p.EmitReadFixed(islice)

					fieldval.Set(slice)
				} else {
					// Length for the slice not specified. Try parsing it as is.
					p.EmitReadFixed(fieldval.Interface())
				}
			case reflect.Struct:
				p.EmitReadStruct(fieldval.Interface())

			}

			/*// Inspect the field's tag to find out how to parse it*/
			/*fieldv := reflect.ValueOf(data).Elem().Field(i)*/
			/*field := t.Elem().Field(i)*/
			/*if fieldv.Kind() == reflect.Struct {*/
			/*p.ReadStructuredData(fieldv.Interface())*/
			/*} else if fieldv.Kind() == reflect.Slice {*/
			/*lenkey := field.Tag.Get("length")*/
			/*var lenfield reflect.Value*/
			/*if len(lenkey) > 0 {*/
			/*lenfield = reflect.ValueOf(data).Elem().FieldByName(lenkey)*/
			/*}*/
			/*flength := int(lenfield.Interface().(uint32))*/
			/*slice := reflect.MakeSlice(field.Type, flength, flength)*/
			/*ss := slice.Interface().([]byte)*/
			/*cg.emitReadSliceByte(ss)*/
			/*// read into slice ...*/
			/*fieldv.Set(slice)*/
			/*}*/
			/*pad := field.Tag.Get("pad")*/
			/*if len(pad) > 0 {*/
			/*// ...*/
			/*}*/

			/*fmt.Printf("Read var-length field %v\n", field.Name)*/
		}
	}
}

func (p *Parser) EmitReadFixed(data interface{}) bool {
	size := binary.Size(data)
	if size < 0 {
		return false
	}

	p.EmitReadFixedFast(data, size)
	return true
}

func (p *Parser) EmitReadFixedFast(data interface{}, size int) {
	err := binary.Read(p.r, p.byteOrder, data)
	if err != nil {
		p.RaiseError(err)
	}
	p.offset += uint32(size)
}

func (p *Parser) EmitReadNBytes(nbytes int) []byte {
	buf := make([]byte, nbytes)
	p.EmitReadFull(buf)
	return buf
}

func (p *Parser) EmitReadFull(buf []byte) {
	nbytes, err := io.ReadFull(p.r, buf)
	if err != nil {
		p.RaiseError(err)
	}
	p.offset += uint32(nbytes)
}

func (p *Parser) RaiseError(err error) {
	panic(err)
}
