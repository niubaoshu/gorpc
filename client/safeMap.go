package client

import (
	"sync"
)

type safeMap struct {
	m       map[uint64]chan []byte
	rwMutex sync.RWMutex
}

func NewSafeMap() *safeMap {
	return &safeMap{m: make(map[uint64]chan []byte)}
}

func (sm *safeMap) Set(key uint64, ch chan []byte) {
	sm.rwMutex.Lock()
	sm.m[key] = ch
	sm.rwMutex.Unlock()
}

func (sm *safeMap) Get(key uint64) (ch chan []byte, ok bool) {
	sm.rwMutex.RLock()
	defer sm.rwMutex.RUnlock()
	ch, ok = sm.m[key]
	return
}

func (sm *safeMap) Del(key uint64) {
	sm.rwMutex.Lock()
	delete(sm.m, key)
	sm.rwMutex.Unlock()
}
