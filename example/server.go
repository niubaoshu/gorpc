package main

import (
	"time"

	"github.com/niubaoshu/gorpc"
)

func main() {
	gorpc.NewServer(Plus, Sub, Echo, Add, Mut, Slow, Now).Start()
}

func Plus(a, b int) int {
	return a + b
}

func Sub(a, b int) int {
	return a - b
}

func Echo(msg string) string {
	return msg
}

func Add(a ...int) (c int) {
	for i := 0; i < len(a); i++ {
		c += a[i]
	}
	return c
}

func Mut(a ...int) (c int) {
	c = 1
	for i := 0; i < len(a); i++ {
		c *= a[i]
	}
	return c
}

func Slow(msg string) string {
	time.Sleep(10 * time.Microsecond)
	return msg
}

func Now() time.Time {
	return time.Now()
}
