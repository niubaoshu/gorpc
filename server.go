package gorpc

import (
	"log"
	"net"
	"reflect"
	"sync"

	"github.com/niubaoshu/gotiny"
)

type (
	server struct {
		funcSum   int
		exitChan  chan struct{}   // notify all goroutines to shutdown
		waitGroup *sync.WaitGroup // wait for all goroutines
		addr      string
		functions []func([]byte) []byte //funcsinfo里存储的数据不会修改
	}
	scall struct {
		dec  *gotiny.Decoder
		enc  *gotiny.Encoder
		vals []reflect.Value
	}
)

const defaultaddr = ":3345"

// Start starts service

func NewServer(funcs ...interface{}) *server {
	fns := make([]func([]byte) []byte, len(funcs))

	for idx, fn := range funcs {
		t := reflect.TypeOf(fn)
		v := reflect.ValueOf(fn)
		inum := t.NumIn()
		itpys := make([]reflect.Type, inum)
		for i := 0; i < inum; i++ {
			itpys[i] = t.In(i)
		}
		onum := t.NumOut()
		otpys := make([]reflect.Type, onum)
		for i := 0; i < onum; i++ {
			otpys[i] = t.Out(i)
		}

		calls := sync.Pool{
			New: func() interface{} {
				ivs := make([]reflect.Value, len(itpys))
				for i := 0; i < len(itpys); i++ {
					ivs[i] = reflect.New(itpys[i]).Elem()
				}
				dec := gotiny.NewDecoderWithTypes(itpys...)
				dec.SetOff(4) //前两个是函数id,后两个存放序列号
				return &scall{
					dec:  dec,
					enc:  gotiny.NewEncoderWithTypes(otpys...),
					vals: ivs,
				}
			},
		}
		if t.IsVariadic() {
			fns[idx] = func(param []byte) []byte {
				c := calls.Get().(*scall)
				c.dec.ResetWith(param)
				c.dec.DecodeValues(c.vals...)
				param[5] = param[3]
				param[4] = param[2]
				param[3] = param[1]
				param[2] = param[0]
				c.enc.ResetWith(param[:6]) //前四个用来存放长度和函数id,后面是序列号
				c.enc.EncodeValues(v.CallSlice(c.vals)...)
				buf := c.enc.Bytes()
				l := len(buf) - 2
				buf[0] = byte(l >> 8)
				buf[1] = byte(l)
				calls.Put(c)
				return buf
			}
		} else {
			fns[idx] = func(param []byte) []byte {
				c := calls.Get().(*scall)
				c.dec.ResetWith(param)
				c.dec.DecodeValues(c.vals...)
				param[5] = param[3]
				param[4] = param[2]
				param[3] = param[1]
				param[2] = param[0]
				c.enc.ResetWith(param[:6]) //前四个用来存放长度和函数id,后面是序列号
				c.enc.EncodeValues(v.Call(c.vals)...)
				buf := c.enc.Bytes()
				l := len(buf) - 2
				buf[0] = byte(l >> 8)
				buf[1] = byte(l)
				calls.Put(c)
				return buf
			}
		}
	}
	s := &server{
		funcSum:   len(funcs),
		exitChan:  make(chan struct{}),
		waitGroup: new(sync.WaitGroup),
		addr:      defaultaddr,
		functions: fns,
	}
	return s
}

func (s *server) Start() {
	addr, err := net.ResolveTCPAddr("tcp", s.addr)
	if err != nil {
		log.Fatalln("地址解析失败", s.addr, err.Error())
	}
	listener, er := net.ListenTCP("tcp", addr)
	if er != nil {
		log.Fatalln("监听端口失败", s.addr, err.Error())
	}
	defer listener.Close()
	go func() {
		<-s.exitChan
		log.Println("Get Stop Command. Now Stoping...")
		if err = listener.Close(); err != nil {
			log.Println(err)
		}
	}()
	log.Println("监听地址", s.addr, "成功")
	for {
		if conn, err := listener.AcceptTCP(); err == nil {
			log.Println("收到连接", conn.RemoteAddr())
			go func() {
				s.waitGroup.Add(1)
				NewConn(conn, s).Start()
				//utils.NewConsume(&Conn{s.exitChan, conn}, &serverHandler{conn: conn}, 10, 40, s.wg).Start()
				s.waitGroup.Done()
			}()
		} else {
			log.Println("连接出错", err.Error())
		}
	}
}

// Stop stops service
func (s *server) Stop() {
	close(s.exitChan)
	s.waitGroup.Wait()
}
