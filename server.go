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
		fnames    []string
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
	l := len(funcs) + 1
	nfs := make([]interface{}, 1, l)
	s := &server{
		funcSum:   l,
		exitChan:  make(chan struct{}),
		waitGroup: new(sync.WaitGroup),
		addr:      defaultaddr,
		fnames:    make([]string, l),
	}
	nfs[0] = func(names []string) []int {
		sl, cl := len(s.fnames), len(names)
		ret := make([]int, cl)
		for i, j := 0, 0; i < sl && j < cl; {
			if s.fnames[i] > names[j] {
				ret[j] = -1 //没有该服务返回0
				j++
			} else if s.fnames[i] < names[j] {
				i++
			} else {
				i++
				ret[j] = i //函数id从1开始
				j++
			}
		}
		return ret
	}
	nfs = append(nfs, funcs...)
	fns := make([]func([]byte) []byte, l)
	for idx, fn := range nfs {
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
				return &scall{
					dec:  dec,
					enc:  gotiny.NewEncoderWithTypes(otpys...),
					vals: ivs,
				}
			},
		}
		var f func([]byte) []byte
		if t.IsVariadic() {
			f = func(param []byte) []byte {
				c := calls.Get().(*scall)
				//0,1个是函数id,1,2个存放序列号
				c.dec.ResetWith(param[4:])
				c.dec.DecodeValues(c.vals...)
				param[5] = param[3]
				param[4] = param[2]
				param[3] = param[1]
				param[2] = param[0]
				c.enc.ResetWithBuf(param[:6]) //前四个用来存放长度和函数id,后面2是序列号
				c.enc.EncodeValues(v.CallSlice(c.vals)...)
				buf := c.enc.Bytes()
				l := len(buf) - 2 // 0,1是保存后面的长度的,所以不计入长度
				buf[0] = byte(l >> 8)
				buf[1] = byte(l)
				calls.Put(c)
				return buf
			}
		} else {
			f = func(param []byte) []byte {
				c := calls.Get().(*scall)
				c.dec.ResetWith(param[4:])
				c.dec.DecodeValues(c.vals...)
				param[5] = param[3]
				param[4] = param[2]
				param[3] = param[1]
				param[2] = param[0]
				c.enc.ResetWithBuf(param[:6]) //前四个用来存放长度和函数id,后面2是序列号
				c.enc.EncodeValues(v.Call(c.vals)...)
				buf := c.enc.Bytes()
				l := len(buf) - 2
				buf[0] = byte(l >> 8)
				buf[1] = byte(l)
				calls.Put(c)
				return buf
			}
		}
		name := findNameWithPtr(v.Pointer())
		for idx > 1 && name < s.fnames[idx-1] {
			s.fnames[idx] = s.fnames[idx-1]
			fns[idx] = fns[idx-1]
			idx--
		}
		s.fnames[idx] = name
		fns[idx] = f
	}
	s.fnames = s.fnames[1:]
	s.functions = fns
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
	log.Println("监听地址", s.addr, "成功,开始提供服务", s.fnames)
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
