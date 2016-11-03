package client

import (
	"fmt"
	"log"
	"net/rpc"
	"testing"
)

type Args struct {
	A, B int
}

var (
	stdCli *rpc.Client
)

func init() {
	var err error
	stdCli, err = rpc.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
}

func StdT() (err error) {
	var reply int
	var msg, rmsg string
	msg = "hello,world"
	if err = stdCli.Call("Stdrpc.Plus", &Args{1, 2}, &reply); err != nil || reply != 3 {
		fmt.Println(err, "1+2=3", reply)
		return err
	}
	if err = stdCli.Call("Stdrpc.Sub", &Args{1, 2}, &reply); err != nil || reply != -1 {
		fmt.Println(err, "1-2=-1", reply)
		return err
	}
	if err = stdCli.Call("Stdrpc.Echo", &msg, &rmsg); err != nil || rmsg != msg {
		fmt.Println(err, "echo hello,world", rmsg)
		return err
	}
	if err = stdCli.Call("Stdrpc.Add", &[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, &reply); err != nil || reply != 55 {
		fmt.Println(err, "1+2+3+4+5+6+7+8+9+10=55", reply)
		return err
	}
	if err = stdCli.Call("Stdrpc.Mut", &[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, &reply); err != nil || reply != 3628800 {
		fmt.Println(err, "1*2*3*4*5*6*7*8*9*10=3628800", reply)
		return err
	}
	if err = stdCli.Call("Stdrpc.Slow", &msg, &rmsg); err != nil || rmsg != msg {
		fmt.Println(err, "echo hello,world", rmsg)
		return err
	}
	return nil
}

func TestStdT(t *testing.T) {
	if err := StdT(); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkStdT(b *testing.B) {
	for i := 0; i < b.N; i++ {
		StdT()
	}
}

func BenchmarkStdTRun(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			StdT()
		}
	})
}
