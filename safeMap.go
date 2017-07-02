package gorpc

import "sync"

type safeMap struct {
	m map[uint64]chan []byte
	sync.RWMutex
}

func (sm *safeMap) set(key uint64, ch chan []byte) {
	sm.Lock()
	sm.m[key] = ch
	sm.Unlock()
}

func (sm *safeMap) get(key uint64) (ch chan []byte, has bool) {
	sm.RLock()
	ch, has = sm.m[key]
	sm.RUnlock()
	return
}

func (sm *safeMap) delhas(key uint64, ch chan []byte) (msg []byte) {
	sm.Lock()
	if len(ch) != 0 {
		msg = <-ch
	}
	delete(sm.m, key)
	sm.Unlock()
	return
}
func (sm *safeMap) del(key uint64) {
	sm.Lock()
	delete(sm.m, key)
	sm.Unlock()
}
func (sm *safeMap) len() int {
	sm.Lock()
	l := len(sm.m)
	sm.Unlock()
	return l
}
