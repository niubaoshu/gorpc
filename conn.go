package gorpc

import (
	"bufio"
	"io"
	"log"
	"net"
	"sync"
	//"time"
)

type Conn struct {
	svr       *server
	conn      *net.TCPConn
	bytesPool sync.Pool
	pool      *workerPool
}

func NewConn(c *net.TCPConn, s *server) (ret *Conn) {
	ret = &Conn{
		conn: c,
		svr:  s,
		bytesPool: sync.Pool{
			New: func() interface{} {
				var arr [32]byte
				return arr[:32]
			},
		},
		pool: &workerPool{
			MaxWorkersCount: 256 * 1024,
		},
	}
	ret.pool.WorkerFunc = ret.handle
	return ret
}

//该函数不会并发的执行，
func (c *Conn) Start() {
	var err error
	var n int
	var packet []byte
	c.pool.Start()
	var length = []byte{0x00, 0x00}
	conn := bufio.NewReader(c.conn)
	for {
		//[]byte{0x00, 0x00} 为心跳包，收到该包重启设置超时时间，并取下一个包
		// for l[0] == 0x00 && l[1] == 0x00 {
		// 	//超过10秒没有收到包超时，关闭连接
		// 	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

		if n, err = io.ReadFull(conn, length); err != nil {
			handleerr(err, packet[:n], c.conn)
			return
		}
		//}

		packet = c.getNByte((int(length[0]) << 8) | int(length[1]))
		if n, err = io.ReadFull(conn, packet); err != nil {
			handleerr(err, packet[:n], c.conn)
			return
		}
		//log.Println("收到", c.conn.RemoteAddr(), "  的数据", length, packet)
		c.pool.Serve(packet)
	}
}

func (c *Conn) handle(packet []byte) {
	funcNum := int(packet[0])<<8 | int(packet[1])
	if funcNum > c.svr.funcSum {
		log.Println("没有要调用的函数", funcNum)
		return
	}
	retbuf := c.svr.functions[funcNum](packet)
	if cap(retbuf) != cap(packet) {
		c.bytesPool.Put(packet)
	}
	if n, err := c.conn.Write(retbuf); err != nil {
		log.Println("向", c.conn.RemoteAddr(), "发送数据失败", retbuf[:n], err.Error())
		return
	}
	//log.Println("向  ", c.conn.RemoteAddr(), "发送数据", retbuf)
	c.bytesPool.Put(retbuf)
}

func (c *Conn) getNByte(n int) []byte {
	b := c.bytesPool.Get().([]byte)
	if len(b) >= n {
		return b[:n]
	}
	return make([]byte, n)
}

func handleerr(err error, packet []byte, c net.Conn) {
	if err == io.EOF {
		c.Close()
		log.Println("client close conn")
		return
	}
	log.Println("从", c.RemoteAddr(), "连接中读取包体时，读到", packet, "数据时发生错误", err.Error())
	c.Close()
}
