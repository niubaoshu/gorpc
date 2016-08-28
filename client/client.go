package client

import (
	"bufio"
	"errors"
	"github.com/niubaoshu/gorpc/gotiny"
	"github.com/niubaoshu/gorpc/utils"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

var (
	perLength = []byte{0x00, 0x00}
	timeout   = errors.New("the revert packet is timeout")
)

type function struct {
	fInNum       int
	seq          *uint64
	safeMap      *safeMap
	chanBytePool sync.Pool
	bytesPool    sync.Pool
}

type Client struct {
	funcsinfo []*function
	addr      string          //端口号
	waitGroup *sync.WaitGroup //等待退出
	conn      *net.TCPConn
}

func NewClient(addr string, funcs []interface{}) *Client {
	funcSum := len(funcs)
	c := &Client{
		addr:      addr,
		waitGroup: new(sync.WaitGroup),
		funcsinfo: make([]*function, funcSum),
	}
	for i, _ := range c.funcsinfo {
		t := reflect.TypeOf(funcs[i])
		seq := uint64(0)
		c.funcsinfo[i] = &function{
			fInNum:  t.NumIn() + 2,
			safeMap: NewSafeMap(),
			chanBytePool: sync.Pool{
				New: func() interface{} {
					return make(chan []byte)
				},
			},
			seq: &seq,
			bytesPool: sync.Pool{
				New: func() interface{} {
					var arr [32]byte
					return arr[:32]
				},
			},
		}
	}
	return c
}

func (c *Client) Start() {
	perLength := []byte{0x00, 0x00}
	addr, err := net.ResolveTCPAddr("tcp", c.addr)
	if err != nil {
		log.Fatalln("地址解析失败", c.addr, err.Error())
	}
	c.conn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Fatalln("链接失败", c.conn.RemoteAddr(), err.Error())
	} else {
		log.Println("连接成功", c.conn.RemoteAddr())
	}
	var n int
	var packet []byte

	go func() {
		d := gotiny.NewDecoder(perLength, 0)
		conn := bufio.NewReader(c.conn)
		for {
			//[]byte{0x00, 0x00} 为心跳包，收到该包重启设置超时时间，并取下一个包
			// for l[0] == 0x00 && l[1] == 0x00 {
			// 	//超过10秒没有收到包超时，关闭连接
			// 	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

			if n, err = io.ReadFull(conn, perLength); err != nil {
				c.conn.Close()
				log.Println("从", c.conn.RemoteAddr(), "连接中读取包头时，读到", perLength[:n], "数据时发送错误", err.Error())
				return
			}
			//}

			packet = utils.GetNByte((int(perLength[0]) << 8) | int(perLength[1])) //耗资源
			if n, err = io.ReadFull(conn, packet); err != nil {
				c.conn.Close()
				log.Println("从", c.conn.RemoteAddr(), "连接中读取包体时，读到", packet[:n], "数据时发生错误", err.Error())
				return
			}
			//log.Println("收到", c.conn.RemoteAddr(), "  的数据", perLength, packet)
			d.SetBuff(packet, 0)
			cha, _ := c.funcsinfo[d.DecUint()].safeMap.Get(d.DecUint())
			cha <- d.Bytes()
		}
	}()

	//go utils.NewConsume(&Conn{exitChan: c.exitChan, conn: c.conn}, &ClientHandler{conn: c.conn, c: c}, 100, 400, c.waitGroup).Start()
}

func (c *Client) Stop() {
	c.waitGroup.Wait()
}

//第一个是uint64类型，函数的id,后面跟原样的参数，再后面跟返回值的指针
func (c *Client) RemoteCall(para ...interface{}) error {
	c.waitGroup.Add(1)
	defer c.waitGroup.Done()

	fid := para[0].(uint64)
	para[0] = &fid
	f := c.funcsinfo[fid]
	seq := atomic.AddUint64(f.seq, 1) //同步
	para[1] = &seq

	ch := f.chanBytePool.Get().(chan []byte) //同步
	//h := make(chan []byte)
	f.safeMap.Set(seq, ch) //同步，设置通道，以便接受
	b := f.bytesPool.Get().([]byte)
	e := gotiny.NewEncoder(b, 2)
	e.Encodes(para[:f.fInNum]...)
	p := e.Bytes()
	if cap(p) != cap(b) {
		f.bytesPool.Put(b)
	}
	length := len(p) - 2
	p[0] = byte(length >> 8)
	p[1] = byte(length)

	if n, err := c.conn.Write(p); err != nil { //同步
		log.Println("向  ", c.conn.RemoteAddr(), "发送数据失败", p[:n], err.Error())
		return err
	}
	f.bytesPool.Put(p)
	//log.Println("向  ", c.conn.RemoteAddr(), "发送数据", p)
	select {
	case b := <-ch:
		f.safeMap.Del(seq)     //同步
		f.chanBytePool.Put(ch) //同步
		gotiny.NewDecoder(b, 0).Decodes(para[f.fInNum:]...)
		utils.BytesPool.Put(b)
		return nil
	case <-time.Tick(5 * time.Second):
		f.safeMap.Del(seq)     //同步
		f.chanBytePool.Put(ch) //同步
		return timeout
	}
}
