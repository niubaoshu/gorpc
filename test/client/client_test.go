package client

import (
	"fmt"
	"testing"
)

func T() (err error) {
	var r int
	var msg string
	if r, err = plus(1, 2); err != nil || r != 3 {
		fmt.Println(err, "1+2=3", r)
		return err
	}
	if r, err = sub(1, 2); err != nil || r != -1 {
		fmt.Println(err, "1-2=-1", r)
		return err
	}
	if msg, err = echo("hello,world"); err != nil || msg != "hello,world" {
		fmt.Println(err, "echo hello,world", msg)
		return err
	}
	if r, err = add(1, 2, 3, 4, 5, 6, 7, 8, 9, 10); err != nil || r != 55 {
		fmt.Println(err, "1+2+3+4+5+6+7+8+9+10=55", r)
		return err
	}
	if r, err = mut(1, 2, 3, 4, 5, 6, 7, 8, 9, 10); err != nil || r != 3628800 {
		fmt.Println(err, "1*2*3*4*5*6*7*8*9*10=3628800", r)
		return err
	}
	if msg, err = slow("hello,world"); err != nil || msg != "hello,world" {
		fmt.Println(err, "echo hello,world", msg)
		return err
	}
	return nil
}
func TestT(t *testing.T) {
	if err := T(); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkT(b *testing.B) {
	for i := 0; i < b.N; i++ {
		T()
	}
}

func BenchmarkTRun(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			T()
		}
	})
}
