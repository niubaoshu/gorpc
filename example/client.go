package main

import (
	"fmt"
	"time"

	"github.com/niubaoshu/gorpc"
)

// 该文件是自动生成的（除了main函数），还没有实现
func main() {
	gorpc.NewClient([]string{"Add", "Echo", "Mut", "Now", "Plus", "Split", "Sub"}, &add, &echo, &mut, &now, &plus, &split, &sub).Start()

	fmt.Println(plus(1, 2))
	fmt.Println(sub(4, 3))
	fmt.Println(echo("sdfsdfsdf"))
	fmt.Println(add(4))
	fmt.Println(mut(5))
	no := time.Now()
	n, _ := now()
	fmt.Println(add(1, 444, 3, 4, 56, 8, 7, 7, 222))
	fmt.Println(no.Sub(n), time.Now().Sub(n))
	fmt.Println(split("1ssss,,2,,4,5,6,77,"))
}

var (
	plus  func(int, int) (int, error)
	sub   func(int, int) (int, error)
	echo  func(string) (string, error)
	add   func(...int) (int, error)
	mut   func(...int) (int, error)
	slow  func(string) (string, error)
	now   func() (time.Time, error)
	split func(string) ([]string, error)
)
