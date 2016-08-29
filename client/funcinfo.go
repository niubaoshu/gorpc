package client

import (
	"sync"
	"reflect"
)

type funcinfo struct {
	fInNum       int
	seq          *uint64
	safeMap      *safeMap
	chanBytePool sync.Pool
	bytesPool    sync.Pool
}

func NewFuncinfo(f interface{})*funcinfo{
	t := reflect.TypeOf(f)
	seq := uint64(0)
	return &funcinfo{
		fInNum:  t.NumIn() + 2,
		safeMap: NewSafeMap(),
		chanBytePool: sync.Pool{
			New: func() interface{} {
				return make(chan []byte)
			},
		},
		seq: &seq,
		bytesPool: sync.Pool{
			New: func() interface{} {
				var arr [32]byte
				return arr[:32]
			},
		},
	}
}