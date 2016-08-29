package gotiny

import (
	"reflect"
	"unsafe"
)

var ()

type encoder struct {
	buff    []byte
	boolBit byte
	boolPos int
	byte10  [10]byte
}

func NewEncoder(b []byte, n int) *encoder {
	return &encoder{buff: b[:n]}
}

func (e *encoder) Bytes() []byte {
	r := e.buff
	e.buff = nil
	e.boolBit = 0
	e.boolPos = 0
	return r
}
func (e *encoder) SetBuff(b []byte, n int) {
	e.buff = b[:n]
}

func (e *encoder) Encodes(in ...interface{}) {
	var v reflect.Value
	for i := 0; i < len(in); i++ {
		v = reflect.ValueOf(in[i])
		if v.Kind() != reflect.Ptr || v.IsNil() { // "nil" or nilpointer is panic
			panic("totiny: cannot encode not pointer type or value  is nil ")
		}
		e.EncodeValue(v.Elem())
	}
}

func (e *encoder) EncodeValues(vs ...reflect.Value) {
	for i := 0; i < len(vs); i++ {
		e.EncodeValue(vs[i])
	}
}

func (e *encoder) EncodeValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		e.EncBool(v.Bool())
	case reflect.Uint8:
		e.EncUint8(uint8(v.Uint()))
	case reflect.Int8:
		e.EncInt8((int8(v.Int())))
	case reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Uintptr:
		e.EncUint(v.Uint())
	case reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		e.EncInt(v.Int())
	case reflect.Float32, reflect.Float64:
		e.EncFloat(v.Float())
	case reflect.Complex64, reflect.Complex128:
		e.EncComplex(v.Complex())
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			e.EncodeValue(v.Index(i))
		}
	case reflect.Map:
		// map zero encode 0x0000
		// if v.IsNil() {
		// 	panic("totiny: cannot encode nil map of type ")
		// }

		l := v.Len()
		e.EncUint(uint64(l))
		keys := v.MapKeys()
		for i := 0; i < l; i++ {
			e.EncodeValue(keys[i])
			e.EncodeValue(v.MapIndex(keys[i]))
		}
	case reflect.Ptr:
		if v.IsNil() {
			panic("totiny: cannot encode nil pointer of type ")
		}
		e.EncodeValue(v.Elem())
	case reflect.Slice:
		//slice zero encode 0x0000 0x0000
		// if v.IsNil() {
		// 	panic("totiny: cannot encode nil slice of type ")
		// }
		l := v.Len()
		e.EncUint(uint64(l))
		e.EncUint(uint64(v.Cap()))
		for i := 0; i < l; i++ {
			e.EncodeValue(v.Index(i))
		}
	case reflect.String:
		l := v.Len()
		e.EncUint(uint64(l))
		e.buff = append(e.buff, []byte(v.String())...)
	case reflect.Struct:
		vt := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if vt.Field(i).PkgPath == "" { // vt.Field(i).PkgPath 等于 ""代表导出字段
				e.EncodeValue(v.Field(i))
			}
		}
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Invalid:
		//panic("暂不支持这些类型")
	}
}

func (e *encoder) EncBool(v bool) {
	if e.boolBit == 0 {
		e.boolBit = 1
		e.boolPos = len(e.buff)
		e.buff = append(e.buff, 0)
	}
	if v {
		e.buff[e.boolPos] |= e.boolBit
	}
	e.boolBit <<= 1
}

func (e *encoder) EncUint8(v uint8) {
	e.buff = append(e.buff, v)
}

func (e *encoder) EncInt8(v int8) {
	e.buff = append(e.buff, uint8(v))
}

func (e *encoder) EncUint(v uint64) {
	e.buff = append(e.buff, e.byte10[:putUvarint(e.byte10[:10], v)]...)
}

func (e *encoder) EncInt(v int64) {
	e.buff = append(e.buff, e.byte10[:putVarint(e.byte10[:10], v)]...)
}

func (e *encoder) EncFloat(v float64) {
	e.EncUint(floatBits(v))
}

func (e *encoder) EncComplex(v complex128) {
	e.EncFloat(real(v))
	e.EncFloat(imag(v))
}

func floatBits(f float64) uint64 {
	u := *((*uint64)(unsafe.Pointer(&f)))
	var v uint64
	for i := 0; i < 8; i++ {
		v <<= 8
		v |= u & 0xFF
		u >>= 8
	}
	return v
}
func putUvarint(buf []byte, x uint64) int {
	i := 0
	for x >= 0x80 {
		buf[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	buf[i] = byte(x)
	return i + 1
}

func putVarint(buf []byte, x int64) int {
	ux := uint64(x) << 1
	if x < 0 {
		ux = ^ux
	}
	return putUvarint(buf, ux)
}

// valid reports whether the value is valid and a non-nil pointer.
// (Slices, maps, and chans take care of themselves.)
func valid(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Invalid:
		return false
	case reflect.Ptr:
		return !v.IsNil()
	}
	return true
}
