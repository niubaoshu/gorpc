package main

import (
	"fmt"
	"time"

	"log"

	"github.com/niubaoshu/gorpc"
)

// 该文件是自动生成的（除了main函数），还没有实现
func main() {
	cli := gorpc.NewClient([]string{"Add", "Echo", "Mut", "Now", "Plus", "Sub"}, &add, &echo, &mut, &now, &plus, &sub)
	err := cli.Start()
	if err != nil {
		log.Fatal(err)
	}
	start := time.Now()
	for i := 0; i < 10; i++ {
		fmt.Println(plus(i, i*i))
		fmt.Println(sub(i, i*i))
		fmt.Println(echo("sdfsdfsdf"))
		fmt.Println(add(i))
		fmt.Println(mut(i))
		fmt.Println(now())
	}
	cli.Stop()
	fmt.Println(time.Now().Sub(start))
}

var (
	plus func(int, int) (int, error)
	sub  func(int, int) (int, error)
	echo func(string) (string, error)
	add  func(...int) (int, error)
	mut  func(...int) (int, error)
	slow func(string) (string, error)
	now  func() (time.Time, error)
)
