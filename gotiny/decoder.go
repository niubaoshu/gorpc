package gotiny

import (
	"reflect"
	"unsafe"
)

const (
	maxVarintLen64 = 10
)

var (
	RNil = reflect.ValueOf(nil)
)

type Decoder struct {
	buf     []byte
	offset  int
	boolBit byte
	boolen  byte
}

//n is offset
func NewDecoder(b []byte) *Decoder {
	return &Decoder{buf: b}
}

func (d *Decoder) GetUnusedBytes() []byte {
	return d.buf[d.offset:]
}

func (d *Decoder) Reset() {
	d.offset = 0
	d.boolen = 0
	d.boolBit = 0
}

func (d *Decoder) ResetWith(b []byte) {
	*d = Decoder{buf: b}
}

func (d *Decoder) Decodes(is ...interface{}) {
	var v reflect.Value
	for _, i := range is {
		v = reflect.ValueOf(i)
		if v.Kind() != reflect.Ptr || v.IsNil() { // must be a ptr but nilptr
			panic("totiny: only decode to pointer type, and not nilpointer")
		}
		d.DecodeValue(v.Elem())
	}
}

func (d *Decoder) DecodeByType(t reflect.Type) (v reflect.Value) {
	v = reflect.New(t).Elem()
	d.DecodeValue(v)
	return
}

func (d *Decoder) DecodeByTypes(ts ...reflect.Type) (vs []reflect.Value) {
	vs = make([]reflect.Value, len(ts))
	for i, t := range ts {
		vs[i] = d.DecodeByType(t)
	}
	return
}

func (d *Decoder) DecodeValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(d.DecBool())
	case reflect.Uint8:
		v.SetUint(uint64(d.DecUint8()))
	case reflect.Int8:
		v.SetInt(int64(d.DecInt8()))
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v.SetUint(d.DecUint())
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(d.DecInt())
	case reflect.Float32, reflect.Float64:
		v.SetFloat(d.DecFloat())
	case reflect.Complex64, reflect.Complex128:
		v.SetComplex(d.DecComplex())
	case reflect.String:
		v.SetString(d.DecString())
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			d.DecodeValue(v.Index(i))
		}
	case reflect.Map:
		if d.DecBool() {
			l := int(d.DecUint())
			if v.IsNil() {
				v.Set(reflect.MakeMap(v.Type()))
			}
			t := v.Type()
			kt, vt := t.Key(), t.Elem()
			for i := 0; i < l; i++ {
				key, val := reflect.New(kt).Elem(), reflect.New(vt).Elem()
				d.DecodeValue(key)
				d.DecodeValue(val)
				v.SetMapIndex(key, val)
			}
		} else {
			if !v.IsNil() {
				v.Set(reflect.New(v.Type()).Elem())
			}
		}
	case reflect.Ptr:
		if d.DecBool() {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			d.DecodeValue(v.Elem())
		} else {
			if !v.IsNil() {
				v.Set(reflect.New(v.Type()).Elem())
			}
		}
	case reflect.Slice:
		if d.DecBool() {
			l := int(d.DecUint())
			if l == 0 || v.Cap() < l {
				v.Set(reflect.MakeSlice(v.Type(), l, l))
			}
			et := v.Type().Elem()
			for i := 0; i < l; i++ {
				elem := reflect.New(et).Elem()
				d.DecodeValue(elem)
				v.Index(i).Set(elem)
			}
			v.Set(v.Slice(0, l))
		} else {
			if !v.IsNil() {
				//v.Set(reflect.NewAt(v.Type(), unsafe.Pointer(uintptr(0))).Elem())
				v.Set(reflect.New(v.Type()).Elem())
			}
		}
	case reflect.Struct:
		fl := v.NumField()
		for i := 0; i < fl; i++ {
			fv := v.Field(i)
			if fv.CanSet() { //导出字段
				d.DecodeValue(fv)
			} else {
				d.DecodeValue(reflect.NewAt(fv.Type(), unsafe.Pointer(fv.UnsafeAddr())).Elem())
			}
		}
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Invalid:
		//panic("暂不支持这些类型")
	}
}

func (d *Decoder) DecBool() (b bool) {
	if d.boolBit == 0 {
		d.boolBit = 1
		d.boolen = d.buf[d.offset]
		d.offset++
	}
	b = d.boolen&d.boolBit != 0
	d.boolBit <<= 1
	return
}

func (d *Decoder) DecUint8() uint8 {
	d.offset++
	return d.buf[d.offset-1]
}

func (d *Decoder) DecInt8() int8 {
	return int8(d.DecUint8())
}

func (d *Decoder) DecUint() uint64 {
	u, n := uvarint(d.buf[d.offset:])
	d.offset += n
	return u
}

func (d *Decoder) DecInt() int64 {
	i, n := varint(d.buf[d.offset:])
	d.offset += n
	return i
}

func (d *Decoder) DecFloat() float64 {
	return float64FromBits(d.DecUint())
}

func (d *Decoder) DecComplex() complex128 {
	real := float64FromBits(d.DecUint())
	imag := float64FromBits(d.DecUint())
	return complex(real, imag)
}

func (d *Decoder) DecString() string {
	l := int(d.DecUint())
	s := string(d.buf[d.offset : d.offset+l])
	d.offset += l
	return s
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
