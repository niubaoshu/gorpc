package test

import (
	"io"
	"net"
	"unsafe"

	"time"

	"github.com/niubaoshu/gorpc"
)

var info = gorpc.GetFnsInfo((*Sdk)(nil))

type Sdk struct {
	fns []func(...unsafe.Pointer) error
	c   *gorpc.Client
}

func (t *Sdk) Add(a, b int) (c int, err error) {
	err = t.fns[0](unsafe.Pointer(&a), unsafe.Pointer(&b), unsafe.Pointer(&c))
	return
}

func (t *Sdk) Now() (time time.Time, err error) {
	err = t.fns[1](unsafe.Pointer(&time))
	return
}

func (t *Sdk) PingPong(msg string) (rmsg string, err error) {
	err = t.fns[2](unsafe.Pointer(&msg), unsafe.Pointer(&rmsg))
	return
}

func NewAndStartIO(rwc io.ReadWriteCloser) (*Sdk, error) {
	c := gorpc.NewClient(info)
	sdk, err := c.StartIO(rwc)
	return (*Sdk)(unsafe.Pointer(sdk)), err
}

func NewAndStartConn(conn net.Conn) (*Sdk, error) {
	c := gorpc.NewClient(info)
	sdk, err := c.StartConn(conn)
	return (*Sdk)(unsafe.Pointer(sdk)), err
}

func NewAndDialTCP(addr string) (*Sdk, error) {
	c := gorpc.NewClient(info)
	sdk, err := c.DialTCP(addr)
	return (*Sdk)(unsafe.Pointer(sdk)), err
}

func NewAndDial(network, addr string) (*Sdk, error) {
	c := gorpc.NewClient(info)
	sdk, err := c.Dial(network, addr)
	return (*Sdk)(unsafe.Pointer(sdk)), err
}

func NewAndStart() (*Sdk, error) {
	c := gorpc.NewClient(info)
	sdk, err := c.Start()
	return (*Sdk)(unsafe.Pointer(sdk)), err
}
