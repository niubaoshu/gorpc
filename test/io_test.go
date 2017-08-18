package test

import (
	"testing"
	"time"

	"github.com/niubaoshu/gorpc"
)

func PingPong(msg string) string { return msg }
func Add(a int)                  { count = count + a }
func Inc()                       { count++ }

var (
	helloworld = "hello,world!"
	count      = 0
)

func init() {
	go gorpc.NewServer(PingPong, Add, time.Now, Inc).Start()
	time.Sleep(time.Second)
	err := gorpc.NewClient([]string{"Add", "Inc", "Now", "PingPong"}, &add, &inc, &now, &pingPong).Start()
	if err != nil {
		panic(err)
	}
}

func TestIO(t *testing.T) {
	msg, err := pingPong(helloworld)
	if err != nil {
		t.Fatal(err)
	} else if msg != helloworld {
		t.Fatal(msg)
	}

	if now, err := now(); err != nil {
		t.Fatal(err)
	} else {
		t.Log(now)
	}

	if err = add(1); err != nil {
		t.Fatal(err)
	} else {
		t.Log(count)
	}

	if err := inc(); err != nil {
		t.Fatal(err)
	} else {
		t.Log(count)
	}
}

func BenchmarkIO(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pingPong(helloworld)
		//err := add(2)
		//if err != nil {
		//	b.Fatal(n, err)
		//} else if n != 3 {
		//	b.Fatal(n, err)
		//}
	}
}

func BenchmarkConIO(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pingPong(helloworld)
			//add(1, 2)
		}
	})
}
