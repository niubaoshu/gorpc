package gotiny

import (
	"reflect"
	"unsafe"
)

type Encoder struct {
	buf     []byte
	boolBit byte
	boolPos int
	offset  int
}

func NewEncoder(b []byte) *Encoder {
	return &Encoder{buf: b, offset: len(b)}
}

func (e *Encoder) Bytes() []byte {
	return e.buf
}

func (e *Encoder) Reset() {
	e.buf = e.buf[:e.offset]
	e.boolBit = 0
	e.boolPos = 0
}

func (e *Encoder) ResetWith(b []byte) {
	e.buf = b
	e.offset = len(b)
	e.boolBit = 0
	e.boolPos = 0
}

func (e *Encoder) Encodes(in ...interface{}) {
	for i := 0; i < len(in); i++ {
		e.EncodeValue(reflect.ValueOf(in[i]))
	}
}

func (e *Encoder) EncodeValues(vs ...reflect.Value) {
	for i := 0; i < len(vs); i++ {
		e.EncodeValue(vs[i])
	}
}

func (e *Encoder) EncodeValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		e.EncBool(v.Bool())
	case reflect.Uint8:
		e.EncUint8(uint8(v.Uint()))
	case reflect.Int8:
		e.EncInt8((int8(v.Int())))
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		e.EncUint(v.Uint())
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		e.EncInt(v.Int())
	case reflect.Float32, reflect.Float64:
		e.EncFloat(v.Float())
	case reflect.Complex64, reflect.Complex128:
		e.EncComplex(v.Complex())
	case reflect.String:
		e.EncString(v.String())
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			e.EncodeValue(v.Index(i))
		}
	case reflect.Map:
		isNotNil := !v.IsNil()
		e.EncBool(isNotNil)
		if isNotNil {
			keys := v.MapKeys()
			e.EncUint(uint64(len(keys)))
			for _, key := range keys {
				e.EncodeValue(key)
				e.EncodeValue(v.MapIndex(key))
			}
		}
	case reflect.Ptr:
		isNotNil := !v.IsNil()
		e.EncBool(isNotNil)
		if isNotNil {
			e.EncodeValue(v.Elem())
		}
	case reflect.Slice:
		isNotNil := !v.IsNil()
		//fmt.Println(isNotNil)
		e.EncBool(isNotNil)
		if isNotNil {
			l := v.Len()
			e.EncUint(uint64(l))
			for i := 0; i < l; i++ {
				e.EncodeValue(v.Index(i))
			}
		}
	case reflect.Struct:
		l := v.NumField()
		for i := 0; i < l; i++ {
			e.EncodeValue(v.Field(i))
		}
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Invalid:
		//panic("暂不支持这些类型")
	}
}

func (e *Encoder) EncBool(v bool) {
	if e.boolBit == 0 {
		e.boolBit = 1
		e.boolPos = len(e.buf)
		e.buf = append(e.buf, 0)
	}
	if v {
		e.buf[e.boolPos] |= e.boolBit
	}
	e.boolBit <<= 1
}

func (e *Encoder) EncUint8(v uint8) {
	e.buf = append(e.buf, v)
}

func (e *Encoder) EncInt8(v int8) {
	e.EncUint8(uint8(v))
}

func (e *Encoder) EncUint(v uint64) {
	i := 0
	for v >= 0x80 {
		e.buf = append(e.buf, byte(v)|0x80)
		v >>= 7
		i++
	}
	e.buf = append(e.buf, byte(v))
}

//int -5 -4 -3 -2 -1 0 1 2 3 4 5 6
//uint 9  7  5  3  1 0 2 4 6 8 10 12
func (e *Encoder) EncInt(v int64) {
	x := uint64(v) << 1
	if v < 0 {
		x = ^x
	}
	e.EncUint(x)
}

func (e *Encoder) EncFloat(v float64) {
	e.EncUint(floatBits(v))
}

func (e *Encoder) EncComplex(v complex128) {
	e.EncFloat(real(v))
	e.EncFloat(imag(v))
}
func (e *Encoder) EncString(v string) {
	e.EncUint(uint64(len(v)))
	e.buf = append(e.buf, v...)
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
