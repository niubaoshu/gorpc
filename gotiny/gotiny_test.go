package gotiny

import (
	"reflect"
	"testing"
	//"time"
)

// Test basic operations in a safe manner.
func TestBasicEncoderDecoder(t *testing.T) {
	type str struct {
		A map[int]map[int]string
		B []bool
		c int
	}
	var a = "234234"
	i := map[int]map[int]string{
		1: map[int]string{
			1: a,
		},
	}
	st := str{A: i, B: []bool{true, false, false, false, false, true, true}}
	stp := &st
	stpp := &stp
	var nilnil []byte
	var vs = []interface{}{
		true,
		int(123),
		int8(123),
		int16(-12345),
		int32(123456),
		int64(-1234567),
		uint(123),
		uint8(123),
		uint16(12345),
		uint32(123456),
		uint64(1234567),
		uintptr(12345678),
		float32(1.2345),
		float64(1.2345678),
		complex64(1.2345 + 2.3456i),
		complex128(1.2345678 + 2.3456789i),
		[]byte("hell，中国人"),
		[][]byte{[]byte("hello"), []byte("world")},
		string("hello,日本国"),
		map[int]string{
			1: "h",
			2: "h",
		},
		i,
		st,
		stp,
		stpp,
		struct{}{}, //time.Now(),
		nilnil,
	}
	//for j := 0; j < 10000; j++ {

	//go func() {
	values := make([]interface{}, len(vs))
	rvalues := make([]reflect.Value, len(vs))
	for i := 0; i < len(vs); i++ {
		vp := reflect.New(reflect.TypeOf(vs[i]))
		rvalues[i] = reflect.ValueOf(vs[i])
		vp.Elem().Set(rvalues[i])
		values[i] = vp.Interface()
	}
	b := make([]byte, 0)
	e := NewEncoder(b, 0)
	e.Encodes(values...)
	b = e.Bytes()
	t.Log(b)
	d := NewDecoder(b, 0)
	for _, value := range vs {
		result := reflect.New(reflect.TypeOf(value))
		d.Decodes(result.Interface())
		t.Logf("%T: expected %v got %v", value, value, result.Elem().Interface())
		if !reflect.DeepEqual(value, value) {
			t.Fatalf("%T: expected %v got %v", value, value, result.Elem().Interface())
		}
	}

	e.SetBuff(b, 0)
	e.EncodeValues(rvalues...)
	b = e.Bytes()
	t.Log(b)
	d = NewDecoder(b, 0)
	for i, rvalue := range rvalues {
		result := d.DecodeByType(rvalue.Type())
		t.Logf("%T: expected %v got %v", vs[i], rvalue.Interface(), result.Interface())
		if !reflect.DeepEqual(rvalue.Interface(), result.Interface()) {
			t.Fatalf("%T: expected %v got %v", vs[i], rvalue, result.Interface())
		}
	}
	//}() }
}
