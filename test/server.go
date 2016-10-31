package main

import (
	"fmt"
	"github.com/niubaoshu/gorpc/server"
	"time"
)

func main() {
	funcs := []interface{}{
		plus,
		sub,
		printMsg,
		add,
		mut,
	}
	s := server.NewServer(":3345", funcs)
	s.Start()
}

func plus(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

func printMsg(msg string) {
	//	fmt.Println(msg)
}
func timeout(msg string) {
	time.Sleep(5 * time.Second)
	fmt.Println(msg)
}
func add(a ...int)(c int) {
	for i := 0; i < len(a); i++ {
		c += a[i]
	}
	return c
}
func mut(a ...int)(c int){
	c=1
	for i:=0;i<len(a);i++{
		c *=a[i]
	}
	fmt.Println(c)
	return c
}