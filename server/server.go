package server

import (
	"log"
	"net"
	"reflect"
	"sync"
)

type function struct {
	fValue     reflect.Value
	fInTypes   []reflect.Type
	isVariadic bool
}

type server struct {
	funcSum   uint64
	exitChan  chan struct{}   // notify all goroutines to shutdown
	waitGroup *sync.WaitGroup // wait for all goroutines
	addr      string
	funcsinfo []*function //funcsinfo里存储的数据不会修改
}

// Start starts service

func NewServer(addr string, funcs []interface{}) *server {
	s := &server{
		funcSum:   uint64(len(funcs)),
		exitChan:  make(chan struct{}),
		waitGroup: new(sync.WaitGroup),
		addr:      addr,
		funcsinfo: make([]*function, len(funcs)),
	}
	for i := range s.funcsinfo {
		t := reflect.TypeOf(funcs[i])
		in := t.NumIn()

		s.funcsinfo[i] = &function{
			fValue:   reflect.ValueOf(funcs[i]),
			fInTypes: make([]reflect.Type, in),
		}
		for j := 0; j < in; j++ {
			s.funcsinfo[i].fInTypes[j] = t.In(j)
		}
		s.funcsinfo[i].isVariadic = t.IsVariadic()
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
	log.Println("监听地址", s.addr, "成功")
	for {
		select {
		case <-s.exitChan:
			return
		default:
		}
		//	listener.SetDeadline(<-time.Tick(time.Second))
		if conn, err := listener.AcceptTCP(); err == nil {
			log.Println("收到连接", conn.RemoteAddr())
			go func() {
				s.waitGroup.Add(1)
				NewConn(conn, s).Start()
				//utils.NewConsume(&Conn{s.exitChan, conn}, &serverHandler{conn: conn}, 10, 40, s.waitGroup).Start()
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

type serverHandler struct {
	svr  *server
	conn *net.TCPConn
}
