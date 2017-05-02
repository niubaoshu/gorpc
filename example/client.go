package main

import (
	"unsafe"

	"fmt"
	"time"

	"github.com/niubaoshu/gorpc"
)

var (
	fns [6]*gorpc.Function
)

func main() {
	cli := gorpc.NewClient(fns[:], plus, sub, echo, add, mut, slow)
	cli.Start()
	start := time.Now()
	for i := 0; i < 1; i++ {
		go plus(i, i*i)
		go sub(i, i*i)
		go echo("sdfsdfsdf")
		go add(i)
		go mut(i)
	}
	cli.Stop()
	fmt.Println(time.Now().Sub(start))
}

func plus(a, b int) (c int, err error) {
	err = (fns[0]).Rcall(unsafe.Pointer(&a), unsafe.Pointer(&b), unsafe.Pointer(&c))
	return
}

func sub(a, b int) (c int, err error) {
	err = (fns[1]).Rcall(unsafe.Pointer(&a), unsafe.Pointer(&b), unsafe.Pointer(&c))
	return
}

func echo(msg string) (rmsg string, err error) {
	err = (fns[2]).Rcall(unsafe.Pointer(&msg), unsafe.Pointer(&rmsg))
	return
}

func add(a ...int) (c int, err error) {
	err = (fns[3]).Rcall(unsafe.Pointer(&a), unsafe.Pointer(&c))
	return
}

func mut(a ...int) (c int, err error) {
	err = (fns[4]).Rcall(unsafe.Pointer(&a), unsafe.Pointer(&c))
	return
}

func slow(msg string) (rmsg string, err error) {
	err = (fns[5]).Rcall(unsafe.Pointer(&msg), unsafe.Pointer(&rmsg))
	return
}
