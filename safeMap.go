package gorpc

import "sync"

type safeMap struct {
	m map[uint64]chan []byte
	*sync.RWMutex
}

func newSafeMap() safeMap {
	return safeMap{
		m:       make(map[uint64]chan []byte),
		RWMutex: new(sync.RWMutex),
	}
}

func (sm *safeMap) set(key uint64, ch chan []byte) {
	sm.Lock()
	sm.m[key] = ch
	sm.Unlock()
}

func (sm *safeMap) get(key uint64) (ch chan []byte) {
	sm.RLock()
	ch = sm.m[key]
	sm.RUnlock()
	return
}

func (sm *safeMap) del(key uint64) {
	sm.Lock()
	delete(sm.m, key)
	sm.Unlock()
}
