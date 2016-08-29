package gotiny

import (
	//	"fmt"
	"reflect"
	"unsafe"
)

const (
	MaxVarintLen64 = 10
)

type decoder struct {
	buff    []byte
	offset  int
	boolBit byte
	boolen  byte
}

//n is offset
func NewDecoder(b []byte, n int) *decoder {
	return &decoder{buff: b, offset: n}
}

func (d *decoder) Bytes() []byte {
	r := d.buff[d.offset:]
	d.buff = nil
	d.offset = 0
	d.boolen = 0
	d.boolBit = 0
	return r
}

func (d *decoder) SetBuff(b []byte, n int) {
	d.buff = b
	d.offset = n
}

func (d *decoder) Decodes(is ...interface{}) {
	var v reflect.Value
	for _, i := range is {
		v = reflect.ValueOf(i)
		if v.Kind() != reflect.Ptr || v.IsNil() { // must be a ptr but nilptr
			panic("totiny: only decode to pointer type, and not nilpointer")
		}
		d.DecodeValue(v.Elem())
	}
}

func (d *decoder) DecodeByType(t reflect.Type) (v reflect.Value) {
	v = reflect.New(t).Elem()
	d.DecodeValue(v)
	return
}

func (d *decoder) DecodeByTypes(ts ...reflect.Type) (vs []reflect.Value) {
	vs = make([]reflect.Value, len(ts))
	for i, t := range ts {
		vs[i] = d.DecodeByType(t)
	}
	return
}

func (d *decoder) DecodeValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(d.DecBool())
	case reflect.Uint8:
		v.SetUint(uint64(d.DecUint8()))
	case reflect.Int8:
		v.SetInt(int64(d.DecInt8()))
	case reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Uintptr:
		v.SetUint(d.DecUint())
	case reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		v.SetInt(d.DecInt())
	case reflect.Float32, reflect.Float64:
		v.SetFloat(d.DecFloat())
	case reflect.Complex64, reflect.Complex128:
		v.SetComplex(d.DecComplex())
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			d.DecodeValue(v.Index(i))
		}
	case reflect.Map:
		l := int(d.DecUint())
		va := reflect.MakeMap(v.Type())
		kt := v.Type().Key()
		vt := v.Type().Elem()
		for i := 0; i < l; i++ {
			key := reflect.New(kt).Elem()
			value := reflect.New(vt).Elem()
			d.DecodeValue(key)
			d.DecodeValue(value)
			va.SetMapIndex(key, value)
		}
		v.Set(va)
	case reflect.Ptr:
		// ev := reflect.New(v.Type().Elem()).Elem()
		// d.DecodeValue(ev)
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		d.DecodeValue(v.Elem())
	case reflect.Slice:
		l := int(d.DecUint())
		c := int(d.DecUint())
		et := v.Type().Elem()
		va := reflect.MakeSlice(v.Type(), l, c)
		for i := 0; i < l; i++ {
			elem := reflect.New(et).Elem()
			d.DecodeValue(elem)
			va.Index(i).Set(elem)
		}
		if l != 0 {
			v.Set(va)
		}
	case reflect.String:
		l := int(d.DecUint())
		v.SetString(string(d.buff[d.offset : d.offset+l]))
		d.offset += l
	case reflect.Struct:
		vt := v.Type()
		for i := 0; i < vt.NumField(); i++ {
			if vt.Field(i).PkgPath == "" { //导出字段
				//ft := vt.Field(i).Type
				//fv := reflect.New(ft).Elem()
				d.DecodeValue(v.Field(i))
				//v.Field(i).Set(fv)
			}
		}
	case reflect.Chan, reflect.Func, reflect.Interface:
		//panic("暂不支持这些类型")
	}
}

func (d *decoder) DecBool() bool {
	if d.boolBit == 0 {
		d.boolBit = 1
		d.boolen = d.buff[d.offset]
		d.offset++
	}
	defer func() {
		d.boolBit <<= 1
	}()
	return d.boolen&d.boolBit != 0
}

func (d *decoder) DecUint8() uint8 {
	d.offset++
	return d.buff[d.offset-1]
}

func (d *decoder) DecInt8() int8 {
	d.offset++
	return int8(d.buff[d.offset-1])
}

func (d *decoder) DecUint() uint64 {
	u, n := uvarint(d.buff[d.offset:])
	d.offset += n
	return u
}

func (d *decoder) DecInt() int64 {
	i, n := varint(d.buff[d.offset:])
	d.offset += n
	return i
}

func (d *decoder) DecFloat() float64 {
	return float64FromBits(d.DecUint())
}

func (d *decoder) DecComplex() complex128 {
	real := float64FromBits(d.DecUint())
	imag := float64FromBits(d.DecUint())
	return complex(real, imag)
}

func float64FromBits(u uint64) float64 {
	var v uint64
	for i := 0; i < 8; i++ {
		v <<= 8
		v |= u & 0xFF
		u >>= 8
	}
	return *((*float64)(unsafe.Pointer(&v)))
}

func uvarint(buf []byte) (uint64, int) {
	var x uint64
	var s uint
	for i, b := range buf {
		if b < 0x80 {
			if i > 9 || i == 9 && b > 1 {
				return 0, -(i + 1) // overflow
			}
			return x | uint64(b)<<s, i + 1
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
	return 0, 0
}

func varint(buf []byte) (int64, int) {
	ux, n := uvarint(buf) // ok to continue in presence of error
	x := int64(ux >> 1)
	if ux&1 != 0 {
		x = ^x
	}
	return x, n
}

// func decAlloc(v reflect.Value) reflect.Value {
// 	for v.Kind() == reflect.Ptr {
// 		if v.IsNil() {
// 			v.Set(reflect.New(v.Type().Elem()))
// 		}
// 		v = v.Elem()
// 	}
// 	return v
// }
