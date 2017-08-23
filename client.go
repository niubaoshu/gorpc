package gorpc

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/niubaoshu/gotiny"
)

var (
	rtError         = reflect.TypeOf((*error)(nil)).Elem()
	ErrTimeout      = errors.New("the revert packet is timeout")
	ErrTimeoutR     = reflect.ValueOf(ErrTimeout).Convert(rtError)
	ErrNoFunc       = reflect.ValueOf(errors.New("the func is not exist on server")).Convert(rtError)
	ErrClientClosed = errors.New("the client is closed")
)

type (
	Client struct {
		wg       sync.WaitGroup //等待退出
		exitChan chan struct{}
		rchan    chan []byte
		sms      []*safeMap
		rwc      io.ReadWriteCloser
		// TODO 报错后返回给当前所有调用中的函数
		ErrHandler func(error)
		bufs       []bytesPool
		fns        []interface{}
		names      []string
	}
)

func NewClient(names []string, fns ...interface{}) *Client {
	l := len(fns)
	for i := 0; i < l; i++ {
		ft := reflect.TypeOf(fns[i])
		if ft.Kind() != reflect.Ptr && ft.Elem().Kind() != reflect.Func {
			panic("params must pointer of function")
		}
	}
	return &Client{
		fns:        fns,
		exitChan:   make(chan struct{}),
		rchan:      make(chan []byte, 100*l),
		ErrHandler: defaultErr,
		names:      names,
	}
}

func (c *Client) StartIO(rwc io.ReadWriteCloser) error {
	c.rwc = rwc
	c.wg.Add(2)
	go c.loopRead(rwc)
	fids, err := c.getFids(c.names)
	if err != nil {
		return err
	}
	max := 0
	for i := 0; i < len(fids); i++ {
		if max < fids[i] {
			max = fids[i]
		}
	}
	l, fns := len(c.fns), c.fns //0号位置不放内容
	c.sms = make([]*safeMap, max+1)
	fmt.Println(c.names, fids, l, max)
	for i := 0; i < l; i++ {
		fmt.Println(i)
		c.sms[fids[i]] = makefunc(fns[i], fids[i], rwc)
	}
	go c.receive()
	return nil
}

func (c *Client) StartConn(conn net.Conn) error {
	return c.StartIO(conn)
}

func (c *Client) DialTCP(addr string) error {
	return c.Dial("tcp", addr)
}

func (c *Client) Dial(network, addr string) error {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return err
	}
	log.Printf("链接     %s 成功.\n", conn.RemoteAddr())
	return c.StartConn(conn)
}

func (c *Client) Start() error {
	return c.DialTCP(defaultaddr)
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
			c.ErrHandler(err)
			break
		}
		pack = bufPool.getNByte(int(perLength[0])<<8 | int(perLength[1])) //耗资源
		if _, err = io.ReadFull(conn, pack); err != nil {
			c.ErrHandler(err)
			break
		}
		//log.Println("client:收到数据", perLength, pack)
		c.rchan <- pack
	}
	c.wg.Done()
}

func (c *Client) receive() {
	m := c.sms
	for pack := range c.rchan {
		if ch, has := m[int(pack[0])<<8|int(pack[1])].get(uint64(pack[2])<<8 | uint64(pack[3])); has {
			ch <- pack
		} else {
			// TODO 没有则说明超时了暂时丢弃。
			bufPool.Put(pack)
		}
	}
}

func (c *Client) Stop() {
	close(c.exitChan)
	m := c.sms
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
}

func (c *Client) getFids(names []string) (fids []int, err error) {
	enc := gotiny.NewEncoder(names)
	enc.AppendTo(make([]byte, 32)[:6])
	buf := enc.EncodePtr(unsafe.Pointer(&names))
	l := len(buf) - 2 //0,1字节不计算长度
	buf[0] = byte(l >> 8)
	buf[1] = byte(l)
	if _, err := c.rwc.Write(buf); err != nil {
		return nil, err
	}
	tick := time.NewTicker(5 * time.Second)
	select {
	case pack := <-c.rchan:
		gotiny.Decodes(pack[4:], &fids)
	case <-tick.C:
		return nil, ErrTimeout
	}
	return
}

func defaultErr(err error) {
	fmt.Println(err)
}

func makefunc(fn interface{}, id int, w io.Writer) (sm *safeMap) {
	var f func([]reflect.Value) []reflect.Value
	fnv, fnt := reflect.ValueOf(fn), reflect.TypeOf(fn).Elem()
	inum, onum := fnt.NumIn(), fnt.NumOut()
	dnum := onum - 1
	otyps := make([]reflect.Type, onum)
	for i := 0; i < onum; i++ {
		otyps[i] = fnt.Out(i)
	}
	var (
		encPool   sync.Pool
		decPool   sync.Pool
		bytespool sync.Pool
		chPool    = sync.Pool{
			New: func() interface{} {
				return make(chan []byte)
			},
		}
		timerPool = sync.Pool{
			New: func() interface{} {
				timer := time.NewTimer(5 * time.Second)
				timer.Stop()
				return timer
			},
		}
		seq   uint64
		ityps []reflect.Type
	)

	if id < 1 {
		f = func([]reflect.Value) []reflect.Value {
			ret := makeRet(otyps)
			ret.rvs[dnum] = ErrNoFunc
			return ret.rvs
		}
		fnv.Elem().Set(reflect.MakeFunc(fnt, f))
		return
	}
	sm = &safeMap{m: make(map[uint64]chan []byte)}

	if dnum > 0 {
		decPool.New = func() interface{} { return gotiny.NewDecoderWithType(otyps[:dnum]...) }
	}

	if inum > 0 {
		ityps = make([]reflect.Type, inum)
		for i := 0; i < inum; i++ {
			ityps[i] = fnt.In(i)
		}
		encPool.New = func() interface{} {
			enc := gotiny.NewEncoderWithType(ityps...)
			buf := [32]byte{0, 0, byte(id >> 8), byte(id), 0, 0}
			enc.AppendTo(buf[:6])
			return enc
		}

	}

	if inum == 0 {
		bytespool.New = func() interface{} { return []byte{0, 4, byte(id >> 8), byte(id), 0, 0} } //0,1字节不计算长度,定长是4
	}

	switch {
	case inum > 0 && dnum > 0:
		f = func(ins []reflect.Value) []reflect.Value {
			enc := encPool.Get().(*gotiny.Encoder)
			buf := enc.EncodeValue(ins...)

			l := len(buf) - 2 //0,1字节不计算长度
			buf[0], buf[1] = byte(l>>8), byte(l)
			seq := atomic.AddUint64(&seq, 1) & 0xFFFF // sync
			buf[4], buf[5] = byte(seq>>8), byte(seq)

			rchan := chPool.Get().(chan []byte)
			sm.set(seq, rchan) // sync

			ret := makeRet(otyps)
			if _, err := w.Write(buf); err != nil {
				encPool.Put(enc)
				sm.del(seq)
				chPool.Put(rchan)
				ret.rvs[dnum] = reflect.ValueOf(err)
				return ret.rvs
			}
			encPool.Put(enc)
			//log.Println("client:发送了  ", buf)

			timer := timerPool.Get().(*time.Timer)
			timer.Reset(5 * time.Second)

			select {
			case b := <-rchan:
				timer.Stop()
				timerPool.Put(timer)
				sm.del(seq) // sync
				chPool.Put(rchan)
				dec := decPool.Get().(*gotiny.Decoder)
				dec.DecodePtr(b[4:], ret.ptrs...)
				decPool.Put(dec)
				bufPool.Put(b)
				return ret.rvs
			case <-timer.C: // 超时
				// TODO 如果rchan 里有值(超时后未删除前放入值了)需要处理
				if b := sm.delhas(seq, rchan); b != nil {
					bufPool.Put(b)
				} // sync
				chPool.Put(rchan)
				timer.Stop()
				timerPool.Put(timer)
				ret.rvs[dnum] = ErrTimeoutR
				return ret.rvs
			}
		}
	case inum > 0:
		f = func(ins []reflect.Value) []reflect.Value {
			enc := encPool.Get().(*gotiny.Encoder)
			buf := enc.EncodeValue(ins...)

			l := len(buf) - 2 //0,1字节不计算长度
			buf[0], buf[1] = byte(l>>8), byte(l)
			seq := atomic.AddUint64(&seq, 1) & 0xFFFF // sync
			buf[4], buf[5] = byte(seq>>8), byte(seq)

			rchan := chPool.Get().(chan []byte)
			sm.set(seq, rchan) // sync

			ret := makeRet(otyps)
			if _, err := w.Write(buf); err != nil {
				encPool.Put(enc)
				sm.del(seq)
				chPool.Put(rchan)
				ret.rvs[dnum] = reflect.ValueOf(err)
				return ret.rvs
			}
			encPool.Put(enc)
			//log.Println("client:发送了", buf)

			timer := timerPool.Get().(*time.Timer)
			timer.Reset(5 * time.Second)
			select {
			case b := <-rchan:
				sm.del(seq) // sync
				bufPool.Put(b)
			case <-timer.C: // 超时
				//TODO 如果rchan 里有值(超市后未删除前放入值了)需要处理
				if b := sm.delhas(seq, rchan); b != nil {
					bufPool.Put(b)
				} // sync
				ret.rvs[dnum] = ErrTimeoutR
			}
			chPool.Put(rchan)
			timer.Stop()
			timerPool.Put(timer)
			return ret.rvs
		}
	case dnum > 0:
		f = func(ins []reflect.Value) []reflect.Value {
			buf := bytespool.Get().([]byte)
			seq := atomic.AddUint64(&seq, 1) & 0xFFFF // sync
			buf[4], buf[5] = byte(seq>>8), byte(seq)

			rchan := chPool.Get().(chan []byte)
			sm.set(seq, rchan) // sync

			ret := makeRet(otyps)
			if _, err := w.Write(buf); err != nil {
				sm.del(seq)
				chPool.Put(rchan)
				bufPool.Put(buf)
				ret.rvs[dnum] = reflect.ValueOf(err)
				return ret.rvs
			}
			//log.Println("client:发送了", buf)
			bytespool.Put(buf)

			timer := timerPool.Get().(*time.Timer)
			timer.Reset(5 * time.Second)
			select {
			case b := <-rchan:
				sm.del(seq) // sync
				chPool.Put(rchan)
				timer.Stop()
				timerPool.Put(timer)
				dec := decPool.Get().(*gotiny.Decoder)
				dec.DecodePtr(b[4:], ret.ptrs...)
				decPool.Put(dec)
				bufPool.Put(b)
				return ret.rvs
			case <-timer.C: // 超时
				//TODO 如果rchan 里有值(超市后未删除前放入值了)需要处理
				if b := sm.delhas(seq, rchan); b != nil {
					bufPool.Put(b)
				} // sync
				chPool.Put(rchan)
				timer.Stop()
				timerPool.Put(timer)
				ret.rvs[dnum] = ErrTimeoutR
				return ret.rvs
			}
		}
	default:
		f = func(ins []reflect.Value) []reflect.Value {
			buf := bytespool.Get().([]byte)
			seq := atomic.AddUint64(&seq, 1) & 0xFFFF // sync
			buf[4], buf[5] = byte(seq>>8), byte(seq)

			rchan := chPool.Get().(chan []byte)
			sm.set(seq, rchan) // sync

			ret := makeRet(otyps)
			if _, err := w.Write(buf); err != nil {
				bufPool.Put(buf)
				sm.del(seq)
				chPool.Put(rchan)
				ret.rvs[dnum] = reflect.ValueOf(err)
				return ret.rvs
			}
			//log.Println("client:发送了", buf)
			bytespool.Put(buf)

			timer := timerPool.Get().(*time.Timer)
			timer.Reset(5 * time.Second)
			select {
			case b := <-rchan:
				sm.del(seq) // sync
				bufPool.Put(b)
			case <-timer.C: // 超时
				//TODO 如果rchan 里有值(超市后未删除前放入值了)需要处理
				if b := sm.delhas(seq, rchan); b != nil {
					bufPool.Put(b)
				} // sync
				ret.rvs[dnum] = ErrTimeoutR
			}
			chPool.Put(rchan)
			timer.Stop()
			timerPool.Put(timer)
			return ret.rvs
		}
	}
	fnv.Elem().Set(reflect.MakeFunc(fnt, f))
	return sm
}

func makeRet(otyps []reflect.Type) vals {
	l := len(otyps)
	rvs, ptrs := make([]reflect.Value, l), make([]unsafe.Pointer, l)
	for i := 0; i < l; i++ {
		rv := reflect.New(otyps[i]).Elem()
		rvs[i], ptrs[i] = rv, unsafe.Pointer(rv.UnsafeAddr())
	}
	return vals{rvs, ptrs[:l-1]}
}
