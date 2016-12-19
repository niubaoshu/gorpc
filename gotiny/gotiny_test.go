package gotiny

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"testing"
	"time"
)

type str struct {
	A map[int]map[int]string
	B []bool
	c int
}

type ET0 struct {
	s str
	F map[int]map[int]string
}

var (
	now = time.Now()
	a   = "234234"
	i   = map[int]map[int]string{
		1: map[int]string{
			1: a,
		},
	}
	st = str{A: i, B: []bool{true, false, false, false, false, true, true, false, true, false, true}, c: 234234}
	//st     = str{c: 234234}
	et0      = ET0{s: st, F: i}
	stp      = &st
	stpp     = &stp
	nilslice []byte
	slice    = []byte{1, 2, 3}
	mapt     = map[int]int{0: 1, 1: 2, 2: 3, 3: 4}
	nilmap   map[int][]byte
	nilptr   *map[int][]string
	vs       = []interface{}{
		true,
		false,
		int(123456),
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
		string("hello,日本国"),
		string("9b899bec35bc6bb8"),
		[][][][3][][3]int{{{{{{2, 3}}}}}},
		[][]map[int]map[int]map[int]int{},
		[]map[int]map[int]map[int]int{{1: {2: {3: 4}}}},
		[][]bool{},
		[]byte("hello，中国人"),
		[][]byte{[]byte("hello"), []byte("world")},
		[4]string{"2324", "23423", "捉鬼", "《：LSESERsef色粉色问问我二维牛"},
		map[int]string{1: "h", 2: "h", 3: "nihao"},
		map[string]map[int]string{"werwer": {1: "呼呼喊喊"}, "汉字": {2: "世界"}},
		a,
		i,
		&i,
		st,
		stp,
		stpp,
		[][][]struct{}{},
		struct {
			a, C int
		}{1, 2},
		et0,
		now,
		nilmap,
		nilptr,
		nilslice,
		slice,
		mapt,
	}
	b = make([]byte, 0, 1024)
	e = NewEncoder(b)
	d = NewDecoder(b)

	rvalues  = make([]reflect.Value, len(vs))
	rtypes   = make([]reflect.Type, len(vs))
	results  = make([]reflect.Value, len(vs))
	presults = make([]interface{}, len(vs))

	buf     = make([]byte, 0, 1024)
	network = bytes.NewBuffer(buf) // Stand-in for a network connection
	//network bytes.Buffer
	enc = gob.NewEncoder(network) // Will write to network.
	dec = gob.NewDecoder(network) // Will read from network.
)

func init() {
	for i := 0; i < len(vs); i++ {
		rtypes[i] = reflect.TypeOf(vs[i])
		rvalues[i] = reflect.ValueOf(vs[i])

		if i == len(vs)-2 {
			a := make([]byte, 5)
			vp := reflect.ValueOf(&a)
			results[i] = vp.Elem()
			presults[i] = vp.Interface()
		} else if i == len(vs)-1 {
			//a := map[int]int{111: 233, 6: 7}
			a := map[int]int{}
			vp := reflect.ValueOf(&a)
			results[i] = vp.Elem()
			presults[i] = vp.Interface()
		} else {
			vp := reflect.New(rtypes[i])
			results[i] = vp.Elem()
			presults[i] = vp.Interface()
		}
	}

	bb := make([]byte, 0, 1024)
	ee := NewEncoder(bb)
	ee.Encodes(vs...)
	fmt.Println("gotiny length:", len(ee.Bytes()))

	// buf := make([]byte, 0, 1024)
	// network := bytes.NewBuffer(buf) // Stand-in for a network connection
	// enc := gob.NewEncoder(network)  // Will write to network.
	// for i := 0; i < len(vs); i++ {
	// 	enc.Encode(vs[i])
	// }
	// fmt.Println("stdgob length:", len(network.Bytes()))
}

// Test basic operations in a safe manner.
func TestBasicEncoderDecoder(t *testing.T) {
	e.Encodes(vs...)
	b := e.Bytes()
	d.ResetWith(b)
	t.Log(b)
	//for i := 0; i < 1000; i++ {
	d.Decodes(presults...)
	d.Reset()
	//}
	for i, result := range results {
		//t.Logf("%T: expected %v got %v ,%T", vs[i], vs[i], result.Interface(), result.Interface())
		if !reflect.DeepEqual(vs[i], result.Interface()) {
			t.Fatalf("%T: expected %#v got %#v ,%T", vs[i], vs[i], result.Interface(), result.Interface())
		}
	}

	//e.SetBuff(b, 0)
	//e.ResetWith(b)
	e.Reset()
	e.EncodeValues(rvalues...)
	//t.Log(e.Bytes())
	//d.SetBuff(b, 0)
	//t.Log(b)

	d.ResetWith(e.Bytes())
	rs := d.DecodeByTypes(rtypes...)
	for i, result := range rs {
		//t.Logf("%T: expected %v got %v ,%T", vs[i], vs[i], result.Interface(), result.Interface())
		if !reflect.DeepEqual(vs[i], result.Interface()) {
			t.Fatalf("%T: expected %#v got %#v ,%T", vs[i], vs[i], result.Interface(), result.Interface())
		}
	}
}

// func BenchmarkStdEncode(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		for j := 0; j < 1000; j++ {
// 			for i := 0; i < len(vs); i++ {
// 				enc.Encode(vs[i])
// 			}
// 		}
// 	}
// }

// func BenchmarkStdDecode(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		for j := 0; j < 1000; j++ {
// 			for i := 0; i < len(presults); i++ {
// 				dec.Decode(presults[i])
// 				//err := dec.Decode(presults[i])
// 				//if err != nil {
// 				//	b.Fatal(j, err.Error())
// 				//}
// 			}
// 		}
// 	}
// }

func BenchmarkEncodes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for i := 0; i < 1000; i++ {
			//e.Reset()
			e.Encodes(vs...)
		}
	}
}

func BenchmarkDecodes(b *testing.B) {
	d.ResetWith(e.Bytes())
	for i := 0; i < b.N; i++ {
		for i := 0; i < 1000; i++ {
			//d.Reset()
			d.Decodes(presults...)
		}
	}
}
