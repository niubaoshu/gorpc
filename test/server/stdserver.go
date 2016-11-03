package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"time"
)

type Stdrpc struct{}

type Args struct {
	A, B int
}

func (t *Stdrpc) Plus(a Args, r *int) error {
	*r = a.A + a.B
	return nil
}

func (t *Stdrpc) Sub(a Args, r *int) error {
	*r = a.A - a.B
	return nil
}

func (t *Stdrpc) Echo(msg string, rmsg *string) error {
	*rmsg = msg
	//fmt.Println(*rmsg, msg)
	return nil
}

func (t *Stdrpc) Add(a []int, r *int) error {
	*r = 0
	for i := 0; i < len(a); i++ {
		*r += a[i]
	}
	return nil
}

func (t *Stdrpc) Mut(a []int, r *int) error {
	*r = 1
	for i := 0; i < len(a); i++ {
		*r *= a[i]
	}
	return nil
}

func (t *Stdrpc) Slow(msg string, rmsg *string) error {
	*rmsg = msg
	//fmt.Println(*rmsg, msg)
	time.Sleep(10 * time.Microsecond)
	return nil
}

func main() {
	sr := new(Stdrpc)
	rpc.Register(sr)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	fmt.Println("正在监听1234端口")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Print("Error: accept rpc connection", err.Error())
			continue
		}
		go rpc.ServeConn(conn)
	}
}
