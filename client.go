package gorpc

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"

	"reflect"

	"unsafe"

	"time"

	"github.com/niubaoshu/gotiny"
)

var (
	perLength = []byte{0x00, 0x00}
	timeout   = errors.New("the revert packet is timeout")
)

type (
	Client struct {
		addr    string         //端口号
		wg      sync.WaitGroup //等待退出
		conn    *net.TCPConn
		schan   chan []byte
		rchan   chan []byte
		fid2map map[int]*safeMap
	}
	Function struct {
		fnId  int
		inum  int
		seq   *uint64
		call  sync.Pool
		schan chan []byte
		*safeMap
	}
	ccall struct {
		enc *gotiny.Encoder
		dec *gotiny.Decoder
		ch  chan []byte
		timer
	}
)

func (f *Function) Rcall(params ...unsafe.Pointer) (err error) {
	c := f.call.Get().(*ccall) // sync

	enc := c.enc
	enc.EncodeByUPtrs(params[:f.inum]...)
	buf := enc.Bytes()
	enc.Reset()

	l := len(buf) - 2
	buf[0] = byte(l >> 8)
	buf[1] = byte(l)
	seq := atomic.AddUint64(f.seq, 1) % 65536 // sync
	buf[4] = byte(seq >> 8)
	buf[5] = byte(seq)

	rch := c.ch
	//chmap := f.rchan
	f.set(seq, rch) // sync

	f.schan <- buf // sync
	c.reset()

	select {
	case b := <-rch:
		c.dec.ResetWith(b)
		c.dec.DecodeByUPtr(params[f.inum:]...)
	case <-c.c:
		err = timeout
	}
	c.stop()
	f.del(seq)    // sync
	f.call.Put(c) // sync
	return
}

func NewClient(funcs []*Function, fns ...interface{}) (c *Client) {
	length := len(fns)
	if length != len(funcs) {
		panic("length must equal")
	}
	c = &Client{
		addr:    defaultaddr,
		schan:   make(chan []byte, length*100),
		rchan:   make(chan []byte, length*100),
		fid2map: make(map[int]*safeMap, length),
	}
	for idx, fn := range fns {
		seq := uint64(0)
		t := reflect.TypeOf(fn)
		inum := t.NumIn()
		itpys := make([]reflect.Type, inum)
		for i := 0; i < inum; i++ {
			itpys[i] = t.In(i)
		}
		onum := t.NumOut() - 1
		otpys := make([]reflect.Type, onum)
		for i := 0; i < onum; i++ {
			otpys[i] = t.Out(i)
		}
		c.fid2map[idx] = &safeMap{
			m: make(map[uint64]chan []byte),
		}
		funcs[idx] = &Function{
			fnId:    idx,
			seq:     &seq,
			inum:    inum,
			schan:   c.schan,
			safeMap: c.fid2map[idx],
			call: func(idx int) sync.Pool {
				return sync.Pool{
					New: func() interface{} {
						dec := gotiny.NewDecoderWithTypes(otpys...)
						dec.SetOff(4) //前两个是函数id,后两个存放序列号
						enc := gotiny.NewEncoderWithTypes(itpys...)
						enc.ResetWith([]byte{0, 0, byte(idx >> 8), byte(idx), 0, 0})
						return &ccall{
							dec:   dec,
							enc:   enc,
							ch:    make(chan []byte),
							timer: newTimer(5 * time.Second),
						}
					},
				}
			}(idx),
		}
	}
	return
}

func (c *Client) Start() error {
	addr, err := net.ResolveTCPAddr("tcp", c.addr)
	if err != nil {
		log.Fatalln("地址解析失败", c.addr, err.Error())
		return err
	}
	c.conn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Fatalln("链接失败", c.addr, err.Error())
		return err
	} else {
		log.Println("连接成功", c.conn.RemoteAddr())
	}
	c.wg.Add(3)
	go c.send()
	go c.receive()
	go c.start()
	return nil
}

func (c *Client) start() {
	var n int
	var err error

	var packet []byte
	perLength := []byte{0x00, 0x00}
	conn := bufio.NewReader(c.conn)
	for {
		//[]byte{0x00, 0x00} 为心跳包，收到该包重启设置超时时间，并取下一个包
		// for l[0] == 0x00 && l[1] == 0x00 {
		// 	//超过10秒没有收到包超时，关闭连接
		// 	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

		if n, err = io.ReadFull(conn, perLength); err != nil {
			c.conn.Close()
			log.Println("从", c.conn.RemoteAddr(), "连接中读取包头时，读到", perLength[:n], "数据时发生错误", err.Error())
			return
		}
		//}

		packet = GetNByte(int(perLength[0])<<8 | int(perLength[1])) //耗资源
		if n, err = io.ReadFull(conn, packet); err != nil {
			c.conn.Close()
			log.Println("从", c.conn.RemoteAddr(), "连接中读取包体时，读到", packet[:n], "数据时发生错误", err.Error())
			return
		}
		//log.Println("收到", c.conn.RemoteAddr(), "  的数据", perLength, packet)
		c.rchan <- packet
	}
}
func (c *Client) send() {
	conn := c.conn
	for pack := range c.schan {
		n, err := conn.Write(pack)
		if err != nil {
			log.Println("向", c.conn.RemoteAddr(), "连接中发送", pack[:n], "时发生错误", err.Error())
		}
		//log.Println("向", c.conn.RemoteAddr(), "连接中发送了", pack[:n])
	}
	c.wg.Done()
}

func (c *Client) receive() {
	m := c.fid2map
	for pack := range c.rchan {
		m[int(pack[0])<<8|int(pack[1])].get(uint64(pack[2])<<8 | uint64(pack[3])) <- pack
	}
	c.wg.Done()
}

func (c *Client) Stop() {
	c.wg.Wait()
}