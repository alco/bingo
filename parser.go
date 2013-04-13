package bingo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strconv"
)

type ParseError struct {
	text string
}

func parseError(msg string) *ParseError {
	return &ParseError{msg}
}

func (err *ParseError) Error() string {
	return err.text
}

type Error struct {
	Text string
}

func (err *Error) Error() string {
	return err.Text
}

type ByteOrder binary.ByteOrder
var BigEndian = binary.BigEndian
var LittleEndian = binary.LittleEndian

type ParseOptions int

const (
	Default ParseOptions = iota << 1
	Strict
	Panicky
)

type Parser struct {
	r         io.Reader
	byteOrder binary.ByteOrder
	offset    uint
	context   interface{}

	Tags map[string]interface{}

	strict    bool
	panicky   bool
}

func NewParser(r io.Reader, byteOrder ByteOrder, options ParseOptions) *Parser {
	p := Parser{r: r, Tags: make(map[string]interface{}), byteOrder: byteOrder}
	if options & Strict != 0 {
		p.strict = true
	}
	if options & Panicky != 0 {
		p.panicky = true
	}
	return &p
}

func (p *Parser) Offset() uint {
	return p.offset
}

func (p *Parser) Context() interface{} {
	return p.context
}


type Verifier interface {
	Verify(*Parser) error
}

func (p *Parser) callVerify(data interface{}) {
	if dat, ok := data.(Verifier); ok {
		/*fmt.Printf(">>>>Verifying %v\n", reflect.TypeOf(data))*/
		err := dat.Verify(p)
		if err != nil {
			p.RaiseError(err)
		}
	} else {
		typ := reflect.TypeOf(data)
		if typ != nil {
			if _, ok := typ.MethodByName("Verify"); ok {
				p.RaiseError2("Type %v has a Verify() method with incorrect signature. Expected: Verify(p *bingo.Parser) error.", typ)
			}
		}
	}
}

func (p *Parser) EmitReadStruct(data interface{}) (err error) {
	if !p.panicky {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(runtime.Error); ok {
					panic(r)
				}

				if _, ok := r.(reflect.ValueError); ok {
					panic(r)
				}

				switch x := r.(type) {
				case error:
					err = x
				case string:
					err = errors.New(x)
				default:
					// This should not be reachable unless there's a bug in the package
					panic(r)
				}
			}
		}()
	}

	p.context = data
	p.emitReadStruct(data)
	return
}

func (p *Parser) emitReadStruct(data interface{}) {
	// Initial sanity checks
	ptrtyp := reflect.TypeOf(data)
	if ptrtyp.Kind() != reflect.Ptr {
		p.RaiseError2("Invalid argument type %v. Expected pointer to a struct.", ptrtyp)
	}
	typ := ptrtyp.Elem()
	if typ.Kind() != reflect.Struct {
		p.RaiseError2("Invalid argument type %v. Expected pointer to a struct.", ptrtyp)
	}

	ptrval := reflect.ValueOf(data)
	val := ptrval.Elem()

	nfields := typ.NumField()
	for fieldIdx := 0; fieldIdx < nfields; fieldIdx++ {
		fieldtyp := typ.Field(fieldIdx)
		fieldval := val.Field(fieldIdx)

		if !p.ifTagSatisfied(fieldtyp, ptrtyp, ptrval) {
			continue
		}

		if len(fieldtyp.PkgPath) > 0 {
			// unexported field. skip it
			if p.strict {
				p.RaiseError2("Unable to parse into '%v %v'. Unexported fields are not supported.", fieldtyp.Name, fieldtyp.Type)
			} else {
				continue
			}
		}

		// Remember current offset to calculate padded bytes after reading
		// current field
		offset := p.offset

		switch fieldval.Kind() {
		case reflect.Struct:
			// Construct a pointer to the given field
			// and pass it to a recursive call
			tptr := reflect.PtrTo(fieldtyp.Type)
			ptr := reflect.New(tptr)
			ptr.Elem().Set(fieldval.Addr())

			p.emitReadStruct(ptr.Elem().Interface())

		case reflect.Slice:
			if lenkey := fieldtyp.Tag.Get("len"); len(lenkey) > 0 {
				if lenkey == "<inf>" {
					// read until EOF
					buf := p.EmitReadAll()
					if len(buf) > 0 {
						p.readSliceFromBytes(fieldval, fieldtyp.Type, buf)
					}
				} else {
					var length int

					if len(lenkey) > 2 && lenkey[len(lenkey)-1] == ')' && lenkey[len(lenkey)-2] == '(' {
						methodname := lenkey[:len(lenkey)-2]
						if meth, ok := ptrtyp.MethodByName(methodname); ok {
							ctxval := reflect.ValueOf(p)
							result := meth.Func.Call([]reflect.Value{ptrval, ctxval})[0]
							length = p.extractInt(result)
						} else {
							p.RaiseError(errors.New(fmt.Sprintf("Method with name %v not found", methodname)))
						}
					} else {
						lenfield := val.FieldByName(lenkey)
						length = p.extractInt(lenfield)
					}

					if length > 0 {
						/*fmt.Printf("Allocating slice of length %v, type %v\n", length, fieldtyp.Type)*/
						slice := reflect.MakeSlice(fieldtyp.Type, length, length)
						islice := slice.Interface()

						size := binary.Size(islice)
						if size < 0 {
							// Varsize type, need to parse each element via recursive call
							for i := 0; i < length; i++ {
								elem := slice.Index(i)
								tptr := reflect.PtrTo(fieldtyp.Type.Elem())
								ptr := reflect.New(tptr)
								ptr.Elem().Set(elem.Addr())

								p.emitReadStruct(ptr.Elem().Interface())
							}
						} else {
							/*fmt.Printf("Size of the field %v is %v\n", fieldtyp.Name, size)*/
							/*fmt.Printf("Read so far %v\n", p.offset)*/
							// Fast path for fixed-size element type
							p.EmitReadFixed(islice)
						}
						fieldval.Set(slice)
					}
				}
			} else if sizekey := fieldtyp.Tag.Get("size"); len(sizekey) > 0 {
				var buf []byte
				if sizekey == "<inf>" {
					// read until EOF
					buf = p.EmitReadAll()
				} else {
					sizefield := val.FieldByName(sizekey)
					size := p.extractInt(sizefield)
					buf = p.EmitReadNBytes(size)
				}
				if len(buf) > 0 {
					p.readSliceFromBytes(fieldval, fieldtyp.Type, buf)
				}
			} else {
				// Length for the slice not specified. Try parsing it as is.
				p.EmitReadFixed(fieldval.Interface())
			}


		case reflect.Ptr:
			p.RaiseError2("Error reading field '%v %v'. Pointer fields are not supported.", fieldtyp.Name, fieldtyp.Type)

		case reflect.Func:
			// Ignore functions

		default:
			// Try to read as fixed data
			tptr := reflect.PtrTo(fieldval.Type())
			ptr := reflect.New(tptr)
			ptr.Elem().Set(fieldval.Addr())

			if !p.EmitReadFixed(ptr.Elem().Interface()) {
				p.RaiseError(errors.New(fmt.Sprintf("Unhandled type %v", fieldval.Kind())))
			}
		}

		padding := p.calculatePadding(fieldtyp, offset)
		if padding > 0 {
			p.EmitSkipNBytes(int(padding))
		}
	}

	p.callVerify(data)
}

func (p *Parser) ifTagSatisfied(fieldtyp reflect.StructField, ptrtyp reflect.Type, ptrval reflect.Value) bool {
	// check for a condition
	ifstr := fieldtyp.Tag.Get("if")
	if len(ifstr) > 0 {
		negate := false
		if ifstr[0] == '!' {
			negate = true
			ifstr = ifstr[1:]
		}
		meth, ok := ptrtyp.MethodByName(ifstr)
		if ok {
			// TODO: check method signature
			ctxval := reflect.ValueOf(p)
			result := meth.Func.Call([]reflect.Value{ptrval, ctxval})[0].Interface().(bool)
			if negate == result {
				// Skip this field
				return false
			}
		}
	}
	return true
}

func (p *Parser) calculatePadding(fieldtyp reflect.StructField, offset uint) uint {
	padstr := fieldtyp.Tag.Get("pad")
	if len(padstr) > 0 {
		padding, err := strconv.ParseUint(padstr, 0, 8)
		if err != nil {
			p.RaiseError2("Invalid value for `pad` tag: %v. Expected an integer.", padstr)
		}

		nbytesRead := p.offset - offset
		mod := nbytesRead % uint(padding)
		if mod != 0 {
			return uint(padding) - mod
		}
	}
	return 0
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
		/*p.RaiseError(err)*/
		p.RaiseError(errors.New(fmt.Sprintf("%v while reading of size %v", err, size)))
	}
	p.offset += uint(size)
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
	p.offset += uint(nbytes)
}

func (p *Parser) EmitReadAll() []byte {
	var buf bytes.Buffer
	nbytes, err := buf.ReadFrom(p.r)
	if err != nil {
		p.RaiseError(err)
	}
	p.offset += uint(nbytes)
	return buf.Bytes()
}

func (p *Parser) EmitSkipNBytes(nbytes int) {
	// FIXME: remove unbounded allocation
	p.EmitReadNBytes(nbytes)
}

func (p *Parser) readSliceFromBytes(val reflect.Value, typ reflect.Type, buf []byte) {
	// Fast path for []byte
	if _, ok := val.Interface().([]byte); ok {
		val.Set(reflect.ValueOf(buf))
		return
	}

	// Create a temporary reader just for this function
	tmp_reader, tmp_offset := p.r, p.offset
	p.r = bytes.NewReader(buf)

	size := uint(len(buf))
	sliceval := val
	bytesRead := uint(0)
	for bytesRead < size {
		offset := p.offset
		elemptr := reflect.New(typ.Elem())

		p.emitReadStruct(elemptr.Interface())
		sliceval = reflect.Append(sliceval, elemptr.Elem())

		bytesRead += uint(p.offset - offset)
	}
	if bytesRead != size {
		p.RaiseError(errors.New("Consistency error: mismatch between block size and total size of elements contained in it"))
	}
	// Assign the newly allocated slice to the original field
	val.Set(sliceval)

	// Restore parser's state
	p.r, p.offset = tmp_reader, tmp_offset
}

func (p *Parser) RaiseError(err error) {
	panic(err)
}

func (p *Parser) RaiseError2(msg string, args ...interface{}) {
	panic(parseError(fmt.Sprintf(msg, args...)))
}

func (p *Parser) extractInt(val reflect.Value) int {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(val.Uint())
	default:
		p.RaiseError(errors.New("Unsupported type for length spec. Only integers are supported."))
	}
	return 0
}

func (p *Parser) extractUint(val reflect.Value) uint {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uint(val.Uint())
	default:
		p.RaiseError(errors.New("Unsupported type for size spec. Only integers are supported."))
	}
	return 0
}
