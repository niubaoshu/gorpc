package main

import (
	"fmt"
	"github.com/niubaoshu/gorpc/client"
	//	"runtime"
	"sync"
	"time"
)

var (
	cli *client.Client
)

func main() {
	funcs := []interface{}{
		plus,
		sub,
		printMsg,
		add,
		mut,
	}
	wait := new(sync.WaitGroup)
	cli = client.NewClient("127.0.0.1:3345", funcs)
	cli.Start()
	fmt.Println(time.Now())
	loopnum := 100000
	wait.Add(loopnum)
	for i := 0; i < loopnum; i++ {
		go func() {
			for j := 0; j < 1; j++ {
				r, err := plus(1, 2)
				if err != nil {
					fmt.Println(err.Error())
				} else {
					if r != 3 {
						fmt.Println("3错了。", r)
					} else {
						//fmt.Println("1+2=", r)
					}
				}
				r, err = sub(1, 2)
				if err != nil {
					fmt.Println(err.Error())
				} else {
					if r != -1 {
						fmt.Println("-1错了。", r)
					} else {
						//fmt.Println("1-2=", r)
					}
				}
				err = printMsg("hello,rpc")
				if err != nil {
					fmt.Println(err.Error())
				}

				r, err = add(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
				if err != nil {
					fmt.Println(err.Error())
				} else {
					if r != 55 {
						fmt.Println("55错了。", r)
					} else {
						//	fmt.Println("1+2+3+4+5+6+7+8+9+10=", r)
					}
				}
				r, err = mut(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
				if err != nil {
					fmt.Println(err.Error())
				} else {
					if r != 55 {
						fmt.Println("55错了。", r)
					} else {
						//	fmt.Println("1+2+3+4+5+6+7+8+9+10=", r)
					}
				}

			}
			wait.Done()
		}()
		//time.Sleep(time.Microsecond)
	}
	wait.Wait()
	fmt.Println(time.Now())
	// var mem runtime.MemStats
	// runtime.ReadMemStats(&mem)
	// fmt.Println(mem)
	// runtime.GC()
	// runtime.GC()
	// runtime.GC()
	// runtime.GC()
	// runtime.GC()
	// runtime.ReadMemStats(&mem)
	// fmt.Println(mem)
	//select {}
}

func plus(a, b int) (c int, err error) {
	err = cli.RemoteCall(uint64(0), 0, &a, &b, &c)
	return
}
func sub(a, b int) (c int, err error) {
	err = cli.RemoteCall(uint64(1), 0, &a, &b, &c)
	return
}
func printMsg(msg string) (err error) {
	err = cli.RemoteCall(uint64(2), 0, &msg)
	return
}

// func timeout(msg string) (err error) {
// 	err = client.RemoteCall(uint64(3), 0, &msg)
// 	return
// }

func add(a ...int) (c int, err error) {
	err = cli.RemoteCall(uint64(3), 0, &a, &c)
	return
}
func mut(a ...int) (c int, err error) {
	err = cli.RemoteCall(uint64(4), 0, &a, &c)
	return
}
