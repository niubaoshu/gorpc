package test

import (
	"log"
	"testing"
	"time"

	"github.com/niubaoshu/gorpc"
)

func PingPong(msg string) string { return msg }
func Add(a, b int) int           { return a + b }

var (
	sdk        *Sdk
	helloworld = "hello,world!"
)

func init() {
	go gorpc.NewServer(PingPong, Add, time.Now).Start()
	time.Sleep(time.Second)
	var err error
	sdk, err = NewAndStart()
	if err != nil {
		log.Fatal(err)
	}
}

func TestIO(t *testing.T) {
	msg, err := sdk.PingPong(helloworld)
	if err != nil {
		t.Fatal(err)
	} else if msg != helloworld {
		t.Fatal(msg)
	}
	c, err := sdk.Add(1, 2)
	if err != nil {
		t.Fatal(err)
	} else if c != 3 {
		t.Fatal(c)
	}
	now, err := sdk.Now()
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(now)
	}
}

func BenchmarkIO(b *testing.B) {
	for i := 0; i < b.N; i++ {
		//pingPong(helloworld)
		n, err := sdk.Add(1, 2)
		if err != nil {
			b.Fatal(n, err)
		} else if n != 3 {
			b.Fatal(n, err)
		}
	}
}

func BenchmarkConIO(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			//pingPong(helloworld)
			sdk.Add(1, 2)
		}
	})
}
