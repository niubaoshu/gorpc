package gorpc

import (
	"testing"
	"time"
	"unsafe"
)

var (
	fns        [2]*Function
	helloworld = "hello,world"
)

func PingPong(msg string) string { return msg }
func Add(a, b int) int           { return a + b }

//func PingPong(msg string) string { time.Sleep(6 * time.Second); return msg }
func pingPong(msg string) (rmsg string, err error) {
	err = (fns[0]).Rcall(unsafe.Pointer(&msg), unsafe.Pointer(&rmsg))
	return
}
func add(a, b int) (c int, err error) {
	err = (fns[1]).Rcall(unsafe.Pointer(&a), unsafe.Pointer(&b), unsafe.Pointer(&c))
	return
}
func init() {
	go NewServer(PingPong, Add).Start()
	time.Sleep(time.Second)
	NewClient(fns[:], pingPong, add).Start()
}

func TestIO(t *testing.T) {
	msg, err := pingPong(helloworld)
	if err != nil {
		//t.Fatal(err)
	} else if msg != helloworld {
		t.Fatal(msg)
	}
	c, err := add(1, 2)
	if err != nil {
		//t.Fatal(err)
	} else if c != 3 {
		t.Fatal(c)
	}
}

func t() {
	pingPong(helloworld)
	add(1, 2)
}

func BenchmarkIO(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pingPong(helloworld)
		add(1, 2)
		//if err != nil {
		//	b.Log(err)
		//} else if msg != helloworld {
		//	b.Fatal(msg)
		//}
	}
}

func BenchmarkConIO(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pingPong(helloworld)
			add(1, 2)
		}
	})
}
