package gotiny

import (
	//	"fmt"
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
		isNotNil := d.DecBool()
		if isNotNil == v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if isNotNil {
			d.DecodeValue(v.Elem())
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
		nf := v.NumField()
		for i := 0; i < nf; i++ {
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
	buf, i := d.buf, d.offset
	if buf[i] < 0x80 {
		d.offset++
		return uint64(buf[i])
	}
	// we already checked the first byte
	x := uint64(buf[i]) - 0x80
	i++

	b := uint64(buf[i])
	i++

	x += b << 7
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 7

	b = uint64(buf[i])
	i++
	x += b << 14
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 14

	b = uint64(buf[i])
	i++
	x += b << 21
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 21

	b = uint64(buf[i])
	i++
	x += b << 28
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 28

	b = uint64(buf[i])
	i++
	x += b << 35
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 35

	b = uint64(buf[i])
	i++
	x += b << 42
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 42

	b = uint64(buf[i])
	i++
	x += b << 49
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 49

	b = uint64(buf[i])
	i++
	x += b << 56
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 56

	b = uint64(buf[i])
	i++
	x += b << 63
done:
	d.offset = i
	return x
}

func (d *Decoder) DecInt() int64 {
	return uintToInt(d.DecUint())
}

func (d *Decoder) DecFloat() float64 {
	return uintToFloat(d.DecUint())
}

func (d *Decoder) DecComplex() complex128 {
	return complex(uintToFloat(d.DecUint()), uintToFloat(d.DecUint()))
}

func (d *Decoder) DecString() (s string) {
	l := int(d.DecUint())
	s = string(d.buf[d.offset : d.offset+l])
	d.offset += l
	return
}
