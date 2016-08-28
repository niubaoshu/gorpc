package server

import (
	"bufio"
	"github.com/niubaoshu/gorpc/gotiny"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	//"time"
)

var perLength = []byte{0x00, 0x00}

type Conn struct {
	svr       *server
	conn      *net.TCPConn
	bytesPool sync.Pool
}

func NewConn(c *net.TCPConn, s *server) *Conn {
	return &Conn{
		conn: c,
		svr:  s,
		bytesPool: sync.Pool{
			New: func() interface{} {
				var arr [32]byte
				return arr[:32]
			},
		},
	}
}

//该函数不会并发的执行，
func (c *Conn) Start() {
	var err error
	var n int
	var packet []byte
	var length = []byte{0x00, 0x00}
	conn := bufio.NewReader(c.conn)
	for {
		//[]byte{0x00, 0x00} 为心跳包，收到该包重启设置超时时间，并取下一个包
		// for l[0] == 0x00 && l[1] == 0x00 {
		// 	//超过10秒没有收到包超时，关闭连接
		// 	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

		if n, err = io.ReadFull(conn, length); err != nil {
			if err == io.EOF {
				c.conn.Close()
				log.Println("client close conn")
				return
			}
			c.conn.Close()
			log.Println("从", c.conn.RemoteAddr(), "连接中读取包头时，读到", length[:n], "数据时发送错误", err.Error())
			return
		}
		//}

		packet = c.getNByte((int(length[0]) << 8) | int(length[1]))
		if n, err = io.ReadFull(conn, packet); err != nil {
			c.conn.Close()
			log.Println("从", c.conn.RemoteAddr(), "连接中读取包体时，读到", packet[:n], "数据时发生错误", err.Error())
			return
		}
		//log.Println("收到", c.conn.RemoteAddr(), "  的数据", length, packet)
		go c.handle(packet)
	}
}

func (c *Conn) handle(packet []byte) {
	d := gotiny.NewDecoder(packet, 0)
	funcNum := d.DecUint()
	seq := d.DecUint()

	if funcNum > c.svr.funcSum {
		log.Println("没有要调用的函数", funcNum)
		return
	}

	// ft := reflect.TypeOf(funcs[funcNum])
	// //log.Println(seq, funcNum)

	// vs := make([]reflect.Value, ft.NumIn())
	// for i := 0; i < ft.NumIn(); i++ {
	// 	//log.Println(ft.NumIn(), i)
	// 	vs[i] = d.DecodeByType(ft.In(i))
	// }
	f := c.svr.funcsinfo[funcNum]

	var out []reflect.Value
	if f.isVariadic {
		out = f.fValue.CallSlice(d.DecodeByTypes(f.fInTypes...))
	} else {
		out = f.fValue.Call(d.DecodeByTypes(f.fInTypes...))
	}
	//utils.BytesPool.Put(packet)
	// for i := 0; i < len(rvs); i++ {
	// 	log.Println(rvs[i].Interface())
	// }
	e := gotiny.NewEncoder(packet, 2)
	e.EncUint(funcNum)
	e.EncUint(seq)
	e.EncodeValues(out...)
	p := e.Bytes()
	if cap(p) != cap(packet) {
		c.bytesPool.Put(packet)
	}

	length := len(p) - 2
	p[0] = byte(length >> 8)
	p[1] = byte(length)
	if n, err := c.conn.Write(p); err != nil {
		log.Println("向", c.conn.RemoteAddr(), "发送数据失败", p[:n], err.Error())
		return
	}
	//log.Println("向  ", c.conn.RemoteAddr(), "发送数据", p, funcNum, seq, len(out))
	c.bytesPool.Put(p)
}

func (c *Conn) getNByte(n int) []byte {
	b := c.bytesPool.Get().([]byte)
	if len(b) >= n {
		return b[:n]
	}
	return make([]byte, n)
}
