package client

import (
	"github.com/niubaoshu/gorpc/client"
)

var (
	cli *client.Client
)

func init() {
	funcs := []interface{}{
		plus,
		sub,
		echo,
		add,
		mut,
		slow,
	}
	cli = client.NewClient("127.0.0.1:3345", funcs)
	cli.Start()
}

func plus(a, b int) (c int, err error) {
	err = cli.RemoteCall(uint64(0), 0, &a, &b, &c)
	return
}

func sub(a, b int) (c int, err error) {
	err = cli.RemoteCall(uint64(1), 0, &a, &b, &c)
	return
}

func echo(msg string) (rmsg string, err error) {
	err = cli.RemoteCall(uint64(2), 0, &msg, &rmsg)
	return
}

func add(a ...int) (c int, err error) {
	err = cli.RemoteCall(uint64(3), 0, &a, &c)
	return
}

func mut(a ...int) (c int, err error) {
	err = cli.RemoteCall(uint64(4), 0, &a, &c)
	return
}

func slow(msg string) (rmsg string, err error) {
	err = cli.RemoteCall(uint64(5), 0, &msg, &rmsg)
	return
}
