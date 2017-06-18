package gorpc

import (
	"sync"
)

const (
	tooBig = 1 << 30
)

type Buffer struct {
	buff    []byte
	scratch [64]byte
}

func (b *Buffer) Reset() {
	if len(b.buff) >= tooBig {
		b.buff = b.scratch[0:0]
	} else {
		b.buff = b.buff[0:0]
	}
}

var BufferPool = sync.Pool{
	New: func() interface{} {
		b := new(Buffer)
		b.buff = b.scratch[0:0]
		return b
	},
}

var BytesPool = sync.Pool{
	New: func() interface{} {
		var arr [32]byte
		return arr[:]
	},
}

func (b *Buffer) Bytes() []byte {
	return b.buff
}

func GetNByte(n int) []byte {
	b := BytesPool.Get().([]byte)
	if len(b) >= n {
		return b[:n]
	}
	BytesPool.Put(b)
	return make([]byte, n)
}

func putByte(b []byte) {
	BytesPool.Put(b)
}
