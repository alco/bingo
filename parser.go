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

type Error struct {
	Text string
}

func (err *Error) Error() string {
	return err.Text
}

type Parser struct {
	Strict    bool
	Panicky   bool
	Tags map[string]interface{}

	r         io.Reader
	byteOrder binary.ByteOrder
	offset    uint32
	context   interface{}
}

func (p *Parser) Offset() uint32 {
	return p.offset
}

func (p *Parser) Context() interface{} {
	return p.context
}

func NewParser(r io.Reader, order binary.ByteOrder) *Parser {
	return &Parser{false, false, make(map[string]interface{}), r, order, 0, nil}
}

type Verifier interface {
	Verify(*Parser) error
}

func (p *Parser) callVerify(data interface{}) error {
	if dat, ok := data.(Verifier); ok {
		/*fmt.Printf(">>>>Verifying %v\n", reflect.TypeOf(data))*/
		err := dat.Verify(p)
		return err
	} else {
		typ := reflect.TypeOf(data)
		if typ != nil {
			if _, ok := typ.MethodByName("Verify"); ok {
				p.RaiseError(&Error{fmt.Sprintf("Type %v has a Verify method with incorrect signature. Expected Verify(p *bingo.Parser) error", typ)})
			}
		}
	}
	return nil
}

func (p *Parser) EmitReadStruct(data interface{}) (err error) {
	if !p.Panicky {
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

	// Assign the context on the first (non-recursive) call
	if p.context == nil {
		p.context = data
	}

	/*// Try fast path for fixed-size data*/
	/*if p.EmitReadFixed(data) {*/
	/*return p.callVerify(data)*/
	/*}*/

	// Start reflecting
	ptrtyp := reflect.TypeOf(data)
	typ := ptrtyp.Elem()
	if typ.Kind() != reflect.Struct {
		p.RaiseError(errors.New(fmt.Sprintf("Expected a pointer to a struct. Got %+v", typ.Kind())))
	}
	ptrval := reflect.ValueOf(data)
	val := ptrval.Elem()

	fieldIdx := 0
	nfields := typ.NumField()
	for fieldIdx < nfields {
		/*pendingBytes := 0*/
		/*firstFixedFieldIdx := fieldIdx*/
		for ; fieldIdx < nfields; fieldIdx++ {
			// check for read condition
			fieldtyp := typ.Field(fieldIdx)
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
						/*fmt.Printf("Skipping field %+v\n", fieldtyp.Name)*/
						continue
					}
				}
			}

			// TODO: actually do bulk reading of fixed-size fields,
			// then iterate over them to see if any of them needs verification
			//
			// If a field depends on the previous field passing verification,
			// user should add an `if` tag
			//
			// Bulk reading will be removed eventually, because encoding/binary
			// runs its own loop over struct fields and doesn't actually read
			// them as one unit

			/*fieldval := val.Field(fieldIdx)*/
			/*if fieldval.Kind() == reflect.Ptr && fieldval.IsNil() {*/
			/*break*/
			/*}*/
			/*iface := val.Field(fieldIdx).Interface()*/
			/*fieldSize := binary.Size(iface)*/
			/*if fieldSize <= 0 {*/
			/*// TODO: examine edge case with size == 0*/
			/*break*/
			/*}*/
			/*pendingBytes += fieldSize*/
			/*}*/

			/*// We can now read `pendingBytes` bytes before proceeding*/
			/*if pendingBytes > 0 {*/
			/*buf := p.EmitReadNBytes(pendingBytes)*/
			/*d := &decoder{order: p.byteOrder, buf: buf, firstField: firstFixedFieldIdx, lastField: fieldIdx}*/
			/*d.value(val)*/
			/*}*/

			/*for ; fieldIdx < nfields; fieldIdx++ {*/
			fieldval := val.Field(fieldIdx)
			/*fieldtyp := typ.Field(fieldIdx)*/

			/*if binary.Size(fieldval.Interface()) > 0 {*/
			/*// TODO: examine edge case with size == 0*/
			/*// Fixed-size field. Time to break out from this loop*/
			/*break*/
			/*}*/

			var padding uint32
			offset := p.offset
			padstr := fieldtyp.Tag.Get("pad")
			if len(padstr) > 0 {
				pad, err := strconv.ParseUint(padstr, 0, 8)
				if err != nil {
					p.RaiseError(err)
				}
				padding = uint32(pad)
			}

			switch fieldval.Kind() {
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

									err := p.EmitReadStruct(ptr.Elem().Interface())
									if err != nil {
										return err
									}
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

			case reflect.Struct:
				if !fieldval.CanAddr() {
					p.RaiseError(errors.New("Value cannot Addr()"))
				}
				// Construct a pointer to the given field
				// and pass it to a recursive call
				tptr := reflect.PtrTo(fieldval.Type())
				ptr := reflect.New(tptr)
				ptr.Elem().Set(fieldval.Addr())

				/*fmt.Printf("Parsing into struct %+v\n", fieldval)*/
				err := p.EmitReadStruct(ptr.Elem().Interface())
				if err != nil {
					return err
				}

			case reflect.Ptr:
				/*if fieldval.IsNil() {*/
				/*val := reflect.New(fieldval.Type().Elem())*/
				/*fmt.Printf("%+v\n", val)*/
				/*} else {*/
				/*p.EmitReadStruct(fieldval.Interface())*/
				/*}*/
				/*fmt.Printf("%+v\n", fieldval.Type())*/
				/*fmt.Printf("%+v\n", fieldval.Elem())*/
				/*fmt.Printf("%+v\n", fieldval.Elem().Kind())*/
				p.RaiseError(errors.New("Unsupported ptr field"))

			case reflect.Func:
				// Ignore functions

			default:
				if len(fieldtyp.PkgPath) > 0 {
					// unexported field. skip it
					if p.Strict {
						p.RaiseError(&Error{fmt.Sprintf("Can't parse into unexported field %v", fieldtyp.Name)})
					} else {
						break
					}
				}

				// Try to read as fixed data
				tptr := reflect.PtrTo(fieldval.Type())
				ptr := reflect.New(tptr)
				ptr.Elem().Set(fieldval.Addr())

				if !p.EmitReadFixed(ptr.Elem().Interface()) {
					p.RaiseError(errors.New(fmt.Sprintf("Unhandled type %v", fieldval.Kind())))
				}
			}

			if padding > 1 {
				nbytesRead := p.offset - offset
				mod := nbytesRead % padding
				if mod != 0 {
					/*fmt.Printf(">>> Apply padding of %+v\n", padding - mod)*/
					p.EmitSkipNBytes(int(padding - mod))
				}
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

	return p.callVerify(data)
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

func (p *Parser) EmitReadAll() []byte {
	var buf bytes.Buffer
	nbytes, err := buf.ReadFrom(p.r)
	if err != nil {
		p.RaiseError(err)
	}
	p.offset += uint32(nbytes)
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

		err := p.EmitReadStruct(elemptr.Interface())
		if err != nil {
			p.RaiseError(err)
		}
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
