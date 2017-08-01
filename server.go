package gorpc

import (
	"log"
	"net"
	"reflect"
	"sync"

	"unsafe"

	"github.com/niubaoshu/gotiny"
)

type (
	server struct {
		funcSum   int
		exitChan  chan struct{}  // notify all goroutines to shutdown
		waitGroup sync.WaitGroup // wait for all goroutines
		addr      string
		fns       []func([]byte) []byte //funcsinfo里存储的数据不会修改
	}
	vals struct {
		rvs  []reflect.Value
		ptrs []unsafe.Pointer
	}
)

const defaultaddr = ":3345"

// Start starts service

func NewServer(funcs ...interface{}) *server {
	l := len(funcs) + 1
	fs, fnames := make([]interface{}, l), make([]string, l)
	fs[0] = func(names []string) []int {
		lns := len(names)
		ret := make([]int, lns)
		for i, j := 1, 0; i < l && j < lns; { // i=1 不要第一个
			if fnames[i] == names[j] {
				ret[j] = i
				i++
				j++
			} else if fnames[i] < names[j] {
				i++
			} else {
				ret[j] = -1 //没有该服务返回-1
				j++
			}
		}
		return ret
	}
	copy(fs[1:], funcs)
	fns := make([]func([]byte) []byte, l)
	for idx, fn := range fs {
		t, v := reflect.TypeOf(fn), reflect.ValueOf(fn)
		inum, onum, call := t.NumIn(), t.NumOut(), v.Call
		var ityps, otyps []reflect.Type
		var decpool, encpool, valpool sync.Pool
		var f func(param []byte) []byte
		if t.IsVariadic() {
			call = v.CallSlice
		}
		switch {
		case inum != 0:
			ityps = make([]reflect.Type, inum)
			for i := 0; i < inum; i++ {
				ityps[i] = t.In(i)
			}
			decpool = sync.Pool{New: func() interface{} { return gotiny.NewDecoderWithType(ityps...) }}
			valpool = sync.Pool{
				New: func() interface{} {
					rvs, ptrs := make([]reflect.Value, inum), make([]unsafe.Pointer, inum)
					for i := 0; i < inum; i++ {
						rv := reflect.New(ityps[i]).Elem()
						rvs[i], ptrs[i] = rv, unsafe.Pointer(rv.UnsafeAddr())
					}
					return &vals{rvs, ptrs}
				},
			}
			fallthrough
		case onum != 0:
			otyps = make([]reflect.Type, onum)
			for i := 0; i < onum; i++ {
				otyps[i] = t.Out(i)
			}
			encpool = sync.Pool{New: func() interface{} { return gotiny.NewEncoderWithType(otyps...) }}
			fallthrough
		case inum != 0 && onum != 0:
			f = func(param []byte) []byte {
				d := decpool.Get().(*gotiny.Decoder)
				vs := valpool.Get().(*vals)
				d.DecodePtr(param[4:], vs.ptrs...) //0,1是fid,2,3是seq
				decpool.Put(d)
				ovs := call(vs.rvs)
				valpool.Put(vs)
				param = param[:6]
				param[5], param[4], param[3], param[2] = param[3], param[2], param[1], param[0]
				e := encpool.Get().(*gotiny.Encoder)
				e.AppendTo(param)
				buf := e.EncodeValue(ovs...)
				encpool.Put(e)
				l := len(buf) - 2
				buf[0], buf[1] = byte(l>>8), byte(l)
				return buf
			}
		case inum != 0:
			f = func(param []byte) []byte {
				d := decpool.Get().(*gotiny.Decoder)
				vs := valpool.Get().(*vals)
				d.DecodePtr(param[4:], vs.ptrs...) // 0,1是fid,2,3是seq
				decpool.Put(d)
				call(vs.rvs)
				valpool.Put(vs)
				param = param[:6]
				param[5], param[4], param[3], param[2], param[1], param[0] = param[3], param[2], param[1], param[0], 0, 4
				return param
			}
		case onum != 0:
			f = func(param []byte) []byte { //param 长度为4
				param = param[:6]
				param[5], param[4], param[3], param[2] = param[3], param[2], param[1], param[0]
				ovs := call(nil)
				e := encpool.Get().(*gotiny.Encoder)
				e.AppendTo(param)
				buf := e.EncodeValue(ovs...)
				encpool.Put(e)
				l := len(buf) - 2
				buf[0], buf[1] = byte(l>>8), byte(l)
				return buf
			}
		default:
			f = func(param []byte) []byte { //param 长度为4
				param = param[:6]
				param[5], param[4], param[3], param[2], param[1], param[0] = param[3], param[2], param[1], param[0], 0, 4
				call(nil)
				return param
			}
		}
		name := firstToUpper(getNameByPtr(v.Pointer()))
		for idx > 1 && name < fnames[idx-1] {
			fnames[idx] = fnames[idx-1]
			fns[idx] = fns[idx-1]
			idx--
		}
		fnames[idx] = name
		fns[idx] = f
	}
	return &server{
		funcSum:  l,
		exitChan: make(chan struct{}),
		addr:     defaultaddr,
		fns:      fns,
	}
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
	log.Println("监听地址", s.addr, "成功,开始提供服务")
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
