package bingo

import (
    "encoding/binary"
    "errors"
    "math"
    "reflect"
)

type coder struct {
	order binary.ByteOrder
	buf   []byte
    firstField int
    lastField int
}

type decoder coder

func (d *decoder) uint8() uint8 {
	x := d.buf[0]
	d.buf = d.buf[1:]
	return x
}

func (d *decoder) uint16() uint16 {
	x := d.order.Uint16(d.buf[0:2])
	d.buf = d.buf[2:]
	return x
}

func (d *decoder) uint32() uint32 {
	x := d.order.Uint32(d.buf[0:4])
	d.buf = d.buf[4:]
	return x
}

func (d *decoder) uint64() uint64 {
	x := d.order.Uint64(d.buf[0:8])
	d.buf = d.buf[8:]
	return x
}

func (d *decoder) int8() int8 { return int8(d.uint8()) }

func (d *decoder) int16() int16 { return int16(d.uint16()) }

func (d *decoder) int32() int32 { return int32(d.uint32()) }

func (d *decoder) int64() int64 { return int64(d.uint64()) }

func (d *decoder) value(v reflect.Value) {
	switch v.Kind() {
	case reflect.Array:
		l := v.Len()
		for i := 0; i < l; i++ {
			d.value(v.Index(i))
		}

	case reflect.Struct:
		t := v.Type()
		for i := d.firstField; i < d.lastField; i++ {
			// Note: Calling v.CanSet() below is an optimization.
			// It would be sufficient to check the field name,
			// but creating the StructField info for each field is
			// costly (run "go test -bench=ReadStruct" and compare
			// results when making changes to this code).
			if v := v.Field(i); v.CanSet() || t.Field(i).Name != "_" {
				d.value(v)
			} else {
				d.skip(v)
			}
		}

	case reflect.Slice:
		l := v.Len()
		for i := 0; i < l; i++ {
			d.value(v.Index(i))
		}

	case reflect.Int8:
		v.SetInt(int64(d.int8()))
	case reflect.Int16:
		v.SetInt(int64(d.int16()))
	case reflect.Int32:
		v.SetInt(int64(d.int32()))
	case reflect.Int64:
		v.SetInt(d.int64())

	case reflect.Uint8:
		v.SetUint(uint64(d.uint8()))
	case reflect.Uint16:
		v.SetUint(uint64(d.uint16()))
	case reflect.Uint32:
		v.SetUint(uint64(d.uint32()))
	case reflect.Uint64:
		v.SetUint(d.uint64())

	case reflect.Float32:
		v.SetFloat(float64(math.Float32frombits(d.uint32())))
	case reflect.Float64:
		v.SetFloat(math.Float64frombits(d.uint64()))

	case reflect.Complex64:
		v.SetComplex(complex(
			float64(math.Float32frombits(d.uint32())),
			float64(math.Float32frombits(d.uint32())),
		))
	case reflect.Complex128:
		v.SetComplex(complex(
			math.Float64frombits(d.uint64()),
			math.Float64frombits(d.uint64()),
		))
	}
}

func (d *decoder) skip(v reflect.Value) {
	n, _ := dataSize(v)
	d.buf = d.buf[n:]
}

// dataSize returns the number of bytes the actual data represented by v occupies in memory.
// For compound structures, it sums the sizes of the elements. Thus, for instance, for a slice
// it returns the length of the slice times the element size and does not count the memory
// occupied by the header.
func dataSize(v reflect.Value) (int, error) {
	if v.Kind() == reflect.Slice {
		elem, err := sizeof(v.Type().Elem())
		if err != nil {
			return 0, err
		}
		return v.Len() * elem, nil
	}
	return sizeof(v.Type())
}

func sizeof(t reflect.Type) (int, error) {
	switch t.Kind() {
	case reflect.Array:
		n, err := sizeof(t.Elem())
		if err != nil {
			return 0, err
		}
		return t.Len() * n, nil

	case reflect.Struct:
		sum := 0
		for i, n := 0, t.NumField(); i < n; i++ {
			s, err := sizeof(t.Field(i).Type)
			if err != nil {
				return 0, err
			}
			sum += s
		}
		return sum, nil

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return int(t.Size()), nil
	}
	return 0, errors.New("invalid type " + t.String())
}

