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

	"github.com/niubaoshu/gotiny"
)

var (
	ErrTimeout      = errors.New("the revert packet is timeout")
	ErrNoFunc       = errors.New("the func is not exist on server")
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
		fis        fnsInfos
	}
	sdk struct {
		fns []func(...unsafe.Pointer) error
		c   *Client
	}

	fnsInfos struct {
		fnum int
		fs   []function
		ns   []string
	}

	function struct {
		inum      int
		onum      int
		ityps     []reflect.Type
		decPool   sync.Pool
		chPool    sync.Pool
		timerPool sync.Pool
		sm        safeMap
	}
)

func getfns(fns []function, fids []int, w io.Writer) []func(...unsafe.Pointer) error {
	//defer func() {
	//	if e := recover(); e != nil {
	//		if er, ok := e.(error); ok {
	//			err = er
	//		} else {
	//			err = fmt.Errorf("%v", e)
	//		}
	//	}
	//}()
	l := len(fns)
	rets := make([]func(...unsafe.Pointer) error, l)
	for i := 0; i < l; i++ {
		fn, id := &fns[i], fids[i]
		if id < 0 {
			rets[i] = func(...unsafe.Pointer) error {
				return ErrNoFunc
			}
			continue
		}
		encs, bufs, seq := sync.Pool{}, sync.Pool{}, uint64(0)
		inum, onum := fn.inum, fn.onum
		decs, chs, timers, sm := &fn.decPool, &fn.chPool, &fn.timerPool, &fns[i].sm
		if inum > 0 {
			encs.New = func() interface{} {
				enc := gotiny.NewEncoderWithType(fn.ityps...)
				buf := [32]byte{0, 0, byte(id >> 8), byte(id), 0, 0}
				enc.AppendTo(buf[:6])
				return enc
			}
		}
		if inum == 0 {
			bufs.New = func() interface{} { return []byte{0, 4, byte(id >> 8), byte(id), 0, 0} } //0,1字节不计算长度,定长是4
		}

		switch {
		case inum > 0 && onum > 0:
			rets[i] = func(params ...unsafe.Pointer) (err error) {
				enc := encs.Get().(*gotiny.Encoder)
				buf := enc.EncodePtr(params[:inum]...)

				l := len(buf) - 2 //0,1字节不计算长度
				buf[0] = byte(l >> 8)
				buf[1] = byte(l)
				seq := atomic.AddUint64(&seq, 1) & 0xFFFF // sync
				buf[4] = byte(seq >> 8)
				buf[5] = byte(seq)

				rchan := chs.Get().(chan []byte)
				sm.set(seq, rchan) // sync

				if _, err = w.Write(buf); err != nil {
					encs.Put(enc)
					return err
				}
				encs.Put(enc)
				//log.Println("client:发送了  ", buf)

				timer := timers.Get().(*time.Timer)
				timer.Reset(5 * time.Second)

				select {
				case b := <-rchan:
					sm.del(seq) // sync
					chs.Put(rchan)
					timer.Stop()
					timers.Put(timer)
					dec := decs.Get().(*gotiny.Decoder)
					dec.DecodePtr(b[4:], params[inum:]...)
					decs.Put(dec)
					bufPool.Put(b)
					return
				case <-timer.C: // 超时
					//TODO 如果rchan 里有值(超市后未删除前放入值了)需要处理
					if b := sm.delhas(seq, rchan); b != nil {
						bufPool.Put(b)
					} // sync
					chs.Put(rchan)
					timer.Stop()
					timers.Put(timer)
					return ErrTimeout
				}
			}
		case inum > 0:
			rets[i] = func(params ...unsafe.Pointer) (err error) {
				enc := encs.Get().(*gotiny.Encoder)
				buf := enc.EncodePtr(params[:inum]...)

				l := len(buf) - 2 //0,1字节不计算长度
				buf[0] = byte(l >> 8)
				buf[1] = byte(l)
				seq := atomic.AddUint64(&seq, 1) & 0xFFFF // sync
				buf[4] = byte(seq >> 8)
				buf[5] = byte(seq)

				rchan := chs.Get().(chan []byte)
				sm.set(seq, rchan) // sync

				if _, err = w.Write(buf); err != nil {
					encs.Put(enc)
					return err
				}
				encs.Put(enc)
				//log.Println("client:发送了", buf)

				timer := timers.Get().(*time.Timer)
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
					err = ErrTimeout
				}
				chs.Put(rchan)
				timer.Stop()
				timers.Put(timer)
				return
			}
		case onum > 0:
			rets[i] = func(params ...unsafe.Pointer) (err error) {
				buf := bufs.Get().([]byte)
				seq := atomic.AddUint64(&seq, 1) & 0xFFFF // sync
				buf[4] = byte(seq >> 8)
				buf[5] = byte(seq)

				rchan := chs.Get().(chan []byte)
				sm.set(seq, rchan) // sync

				if _, err = w.Write(buf); err != nil {
					bufs.Put(buf)
					return err
				}
				//log.Println("client:发送了", buf)
				bufs.Put(buf)

				timer := fn.timerPool.Get().(*time.Timer)
				timer.Reset(5 * time.Second)

				select {
				case b := <-rchan:
					sm.del(seq) // sync
					chs.Put(rchan)
					timer.Stop()
					timers.Put(timer)
					dec := decs.Get().(*gotiny.Decoder)
					dec.DecodePtr(b[4:], params[inum:]...)
					decs.Put(dec)
					bufPool.Put(b)
					return
				case <-timer.C: // 超时
					//TODO 如果rchan 里有值(超市后未删除前放入值了)需要处理
					if b := sm.delhas(seq, rchan); b != nil {
						bufPool.Put(b)
					} // sync
					chs.Put(rchan)
					timer.Stop()
					timers.Put(timer)
					return ErrTimeout
				}
			}
		default:
			rets[i] = func(params ...unsafe.Pointer) (err error) {
				buf := bufs.Get().([]byte)
				seq := atomic.AddUint64(&seq, 1) & 0xFFFF // sync
				buf[4] = byte(seq >> 8)
				buf[5] = byte(seq)

				rchan := chs.Get().(chan []byte)
				sm.set(seq, rchan) // sync

				if _, err = w.Write(buf); err != nil {
					bufs.Put(buf)
					return err
				}
				//log.Println("client:发送了", buf)
				bufs.Put(buf)

				timer := timers.Get().(*time.Timer)
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
					err = ErrTimeout
				}
				chs.Put(rchan)
				timer.Stop()
				timers.Put(timer)
				return
			}
		}
	}
	return rets
}

func NewClient(fis fnsInfos) *Client {
	return &Client{
		exitChan:   make(chan struct{}),
		rchan:      make(chan []byte, fis.fnum*100),
		ErrHandler: func(err error) { log.Println(err) },
		fis:        fis,
	}
}
func (c *Client) StartIO(rwc io.ReadWriteCloser) (*sdk, error) {
	c.wg.Add(2)
	c.rwc = rwc
	go c.loopRead(rwc)
	fids, err := c.getFids(c.fis.ns)
	if err != nil {
		return nil, err
	}
	go c.receive()
	s := &sdk{
		fns: getfns(c.fis.fs, fids, rwc),
		c:   c,
	}
	l := c.fis.fnum
	fs := c.fis.fs
	max := 0
	for i := 0; i < l; i++ {
		if fids[i] > max {
			max = fids[i]
		}
	}
	c.sms = make([]*safeMap, max+1) //0号位置不放内容
	sms := c.sms
	for i := 0; i < l; i++ {
		if fids[i] < 1 {
			break
		}
		sms[fids[i]] = &fs[i].sm
	}
	return s, nil
}

func GetFnsInfo(sdk interface{}) fnsInfos {
	t := reflect.TypeOf(sdk)
	nm := t.NumMethod()
	fs, ns := make([]function, nm), make([]string, nm)
	for i := 0; i < nm; i++ {
		m := t.Method(i)
		mt := m.Type
		ns[i] = m.Name
		inum, onum := mt.NumIn()-1, mt.NumOut()-1 // 方法拥有者不参与编码,error 不参与解码
		f := function{
			inum:   inum,
			onum:   onum,
			chPool: sync.Pool{New: func() interface{} { return make(chan []byte) }},
			timerPool: sync.Pool{
				New: func() interface{} {
					timer := time.NewTimer(5 * time.Second)
					timer.Stop()
					return timer
				},
			},
		}
		if inum != 0 {
			ityps := make([]reflect.Type, inum)
			for i := 0; i < inum; i++ {
				ityps[i] = mt.In(i + 1) // 方法拥有者不参与编码
			}
			f.ityps = ityps
		}
		if onum != 0 {
			otpys := make([]reflect.Type, onum)
			for i := 0; i < onum; i++ {
				otpys[i] = mt.Out(i)
			}
			f.decPool.New = func() interface{} { return gotiny.NewDecoderWithType(otpys...) }
		}
		f.sm.m = make(map[uint64]chan []byte)
		fs[i] = f
	}
	return fnsInfos{nm, fs, ns}
}

func (c *Client) StartConn(conn net.Conn) (*sdk, error) {
	return c.StartIO(conn)
}

func (c *Client) DialTCP(addr string) (*sdk, error) {
	return c.Dial("tcp", addr)
}

func (c *Client) Dial(network, addr string) (*sdk, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	log.Printf("链接     %s 成功.\n", conn.RemoteAddr())
	return c.StartConn(conn)
}

func (c *Client) Start() (*sdk, error) {
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
			// TODO 没有说明超时了暂时丢弃。
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
	enc.AppendTo([]byte{0, 0, 0, 0, 0, 0})
	buf := enc.EncodePtr(unsafe.Pointer(&names))
	l := len(buf) - 2 //0,1字节不计算长度
	buf[0] = byte(l >> 8)
	buf[1] = byte(l)
	if _, err = c.rwc.Write(buf); err != nil {
		return nil, err
	}
	tick := time.NewTicker(5 * time.Second)
	select {
	case pack := <-c.rchan:
		gotiny.Decodes(pack[4:], &fids)
	case <-tick.C:
		err = ErrTimeout
	}
	return
}
