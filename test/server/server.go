package main

import (
	"github.com/niubaoshu/gorpc/server"
    "flag"
    "runtime/pprof"
    "os"
    "log"
	"time"
)
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	funcs := []interface{}{
		plus,
		sub,
		echo,
		add,
		mut,
		slow,
	}
    flag.Parse()
    if *cpuprofile != "" {
                f, err := os.Create(*cpuprofile)
                if err != nil {
                                log.Fatal(err)
                                        
                }
                        pprof.StartCPUProfile(f)
                                defer pprof.StopCPUProfile()
                                    
    }
	server.NewServer(":3345", funcs).Start()
}

func plus(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

func echo(msg string) string {
	return msg
}

func add(a ...int) (c int) {
	for i := 0; i < len(a); i++ {
		c += a[i]
	}
	return c
}

func mut(a ...int) (c int) {
	c = 1
	for i := 0; i < len(a); i++ {
		c *= a[i]
	}
	return c
}

func slow(msg string) string {
	time.Sleep(10 * time.Microsecond)
	return msg
}
