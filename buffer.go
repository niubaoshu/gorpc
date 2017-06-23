package gorpc

import (
	"io"
	"sync"
)

type bytesPool struct {
	sync.Pool
}

func newbytesPool() *bytesPool {
	return &bytesPool{
		Pool: sync.Pool{
			New: func() interface{} {
				var arr [32]byte
				return arr[:]
			},
		},
	}
}

func (p *bytesPool) getNByte(n int) []byte {
	b := p.Get().([]byte)
	if cap(b) >= n {
		return b[:n]
	}
	p.Put(b)
	return make([]byte, n)
}

type bufReader struct {
	err  error
	rd   io.Reader
	buf  []byte
	r, w int
}

func newBufReaderSize(rd io.Reader, n int) *bufReader {
	return &bufReader{rd: rd, buf: make([]byte, n)}
}

const defaultBufSize = 4096

func newBufReader(rd io.Reader) *bufReader {
	return newBufReaderSize(rd, defaultBufSize)
}

func (b *bufReader) readErr() (err error) {
	err = b.err
	b.err = nil
	return
}

func (b *bufReader) Read(p []byte) (n int, err error) {
	n = len(p)
	if n == 0 {
		return 0, nil
	}
	if b.r == b.w {
		if b.err != nil {
			return 0, b.readErr()
		}
		if n >= len(b.buf) {
			return b.rd.Read(p)
		}
		b.r, b.w = 0, 0
		n, b.err = b.rd.Read(b.buf)
		if n == 0 {
			return 0, b.readErr()
		}
		b.w += n
	}

	// copy as much as we can
	n = copy(p, b.buf[b.r:b.w])
	b.r += n
	return n, nil
}
