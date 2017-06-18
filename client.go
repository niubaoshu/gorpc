package gorpc

import (
	"errors"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"sort"

	"fmt"

	"github.com/niubaoshu/gotiny"
)

var (
	ErrTimeout      = errors.New("the revert packet is timeout")
	ErrNoFunc       = errors.New("the func is not exist on server")
	ErrClientClosed = errors.New("the client is closed")
	getIdxFunc      *Function
)

type (
	Client struct {
		fnum       int
		wg         sync.WaitGroup //等待退出
		exitChan   chan struct{}
		rchan      chan []byte
		chmap      []*safeMap
		fns        []Function
		rwc        io.ReadWriteCloser
		errHandler func(error)
		*bytesPool
	}
	Function struct {
		ityps     []reflect.Type
		name      string
		fid       int
		inum      int
		seq       uint64
		encPool   sync.Pool
		decPool   sync.Pool
		timerPool sync.Pool
		writer    io.Writer
		exitChan  chan struct{}
		*bytesPool
		*safeMap
	}
)

func (f *Function) Rcall(params ...unsafe.Pointer) (err error) {
	//defer func() {
	//	if e := recover(); e != nil {
	//		if er, ok := e.(error); ok {
	//			err = er
	//		} else {
	//			err = fmt.Errorf("%v", e)
	//		}
	//	}
	//}()

	if f.fid < 0 {
		return ErrNoFunc
	}
	select {
	case <-f.exitChan:
		return ErrClientClosed
	default:
	}

	timer := f.timerPool.Get().(*time.Timer)
	timer.Reset(5 * time.Second)

	enc := f.encPool.Get().(*gotiny.Encoder)
	enc.EncodeByUPtrs(params[:f.inum]...)
	buf := enc.Bytes()
	enc.Reset()

	l := len(buf) - 2 //0,1字节不计算长度
	buf[0] = byte(l >> 8)
	buf[1] = byte(l)
	seq := atomic.AddUint64(&f.seq, 1) & 0xFFFF // sync
	buf[4] = byte(seq >> 8)
	buf[5] = byte(seq)

	rchan := make(chan []byte)
	f.set(seq, rchan) // sync

	if _, err = f.writer.Write(buf); err != nil {
		return err
	}
	f.encPool.Put(enc)
	//log.Println("client:发送了", buf)

	select {
	case b := <-rchan:
		timer.Stop()
		f.timerPool.Put(timer)
		dec := f.decPool.Get().(*gotiny.Decoder)
		dec.ResetWith(b[4:])
		dec.DecodeByUPtr(params[f.inum:]...)
		f.decPool.Put(dec)
		f.Put(b)
	case <-timer.C:
		timer.Stop()
		f.timerPool.Put(timer)
		err = ErrTimeout
	}
	f.del(seq) // sync
	return
}

func NewClient(fns []Function) *Client {
	l := len(fns)
	c := &Client{
		fnum:      l,
		rchan:     make(chan []byte, l*100),
		bytesPool: fns[0].bytesPool,
		exitChan:  fns[0].exitChan,
		fns:       fns,
	}
	return c
}

func NewFuncs(fns ...interface{}) []Function {
	l := len(fns)
	funcs := make([]Function, l)
	bytesPool := newbytesPool()
	exitChan := make(chan struct{})
	for i := 0; i < l; i++ {
		t := reflect.TypeOf(fns[i])
		inum := t.NumIn()
		ityps := make([]reflect.Type, inum)
		for i := 0; i < inum; i++ {
			ityps[i] = t.In(i)
		}
		onum := t.NumOut() - 1 //error 不参与解码
		otpys := make([]reflect.Type, onum)
		for i := 0; i < onum; i++ {
			otpys[i] = t.Out(i)
		}
		funcs[i].safeMap = newSafeMap()
		funcs[i].exitChan = exitChan
		funcs[i].bytesPool = bytesPool
		funcs[i].name = getFuncName(fns[i])
		funcs[i].inum = inum
		funcs[i].ityps = ityps
		funcs[i].decPool = sync.Pool{
			New: func() interface{} { return gotiny.NewDecoderWithTypes(otpys...) },
		}
		funcs[i].timerPool = sync.Pool{
			New: func() interface{} {
				timer := time.NewTimer(5 * time.Second)
				timer.Stop()
				return timer
			},
		}

	}
	return funcs
}

func (c *Client) StartIO(rwc io.ReadWriteCloser) error {

	if c.errHandler == nil {
		c.errHandler = func(err error) { log.Println(err) }
	}
	c.rwc = rwc
	for i := 0; i < c.fnum; i++ {
		c.fns[i].writer = rwc
	}
	c.wg.Add(1)
	go c.loopRead(rwc)

	getIdxFunc = &NewFuncs(getFids)[0]
	go func() {
		p := <-c.rchan
		getIdxFunc.safeMap.get(1) <- p
	}()
	getIdxFunc.encPool = sync.Pool{
		New: func() interface{} {
			enc := gotiny.NewEncoderWithTypes(getIdxFunc.ityps...)
			enc.ResetWithBuf([]byte{0, 0, 0, 0, 0, 0})
			return enc
		},
	}
	getIdxFunc.writer = rwc

	names := make([]string, c.fnum)
	f2name := make(map[string]*Function)
	for i := 0; i < c.fnum; i++ {
		name := c.fns[i].name
		names[i] = name
		f2name[name] = &c.fns[i]
	}
	sort.Strings(names)

	fids, maxid, err := getFids(names)
	if err != nil {
		return err
	}
	c.chmap = make([]*safeMap, maxid+1) // 0位置不提供服务
	for i := 0; i < c.fnum; i++ {
		f := f2name[names[i]]
		f.fid = fids[i]
		if f.fid > 0 {
			c.chmap[f.fid] = f.safeMap
			f.encPool = sync.Pool{
				New: func() interface{} {
					enc := gotiny.NewEncoderWithTypes(f.ityps...)
					enc.ResetWithBuf([]byte{0, 0, byte(f.fid >> 8), byte(f.fid), 0, 0})
					return enc
				},
			}
		}
	}
	go c.receive()
	return nil
}

func (c *Client) StartConn(conn net.Conn) error {
	return c.StartIO(conn)
}

func (c *Client) StartAddr(addr string) error {
	a, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, a)
	if err != nil {
		return err
	}
	log.Printf("链接     %s 成功.\n", conn.RemoteAddr())
	return c.StartConn(conn)
}

func (c *Client) Start() error {
	return c.StartAddr(defaultaddr)
}

func (c *Client) loopRead(r io.Reader) {
	var (
		err       error
		pack      []byte
		perLength = []byte{0x00, 0x00}
		conn      = newBufReader(r)
	)
	for {
		if _, err = io.ReadFull(conn, perLength); err != nil {
			c.errHandler(err)
			break
		}
		pack = c.getNByte(int(perLength[0])<<8 | int(perLength[1])) //耗资源
		if _, err = io.ReadFull(conn, pack); err != nil {
			c.errHandler(err)
			break
		}
		//log.Println("client:收到数据", perLength, pack)
		c.rchan <- pack
	}
	c.wg.Done()
}

func (c *Client) receive() {
	m := c.chmap
	for pack := range c.rchan {
		m[int(pack[0])<<8|int(pack[1])].get(uint64(pack[2])<<8 | uint64(pack[3])) <- pack
	}
	c.wg.Done()
}

func (c *Client) Stop() {
	close(c.exitChan)
	m := c.chmap
	// 检测是否还有未返回的包
	for i := 0; i < len(m) && m[i] != nil; {
		//TODO 检测长度时Rcall 尚未set,测试会发现长度为0，而尚有调用未结束
		if m[i].len() != 0 {
			time.Sleep(100 * time.Microsecond)
		} else {
			i++
		}
	}
	c.rwc.Close()
	c.wg.Wait()
	fmt.Printf("%+v", c.wg)
}

func getFids(names []string) (fids []int, max int, err error) {
	err = getIdxFunc.Rcall(unsafe.Pointer(&names), unsafe.Pointer(&fids), unsafe.Pointer(&max))
	return
}
