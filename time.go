package gorpc

import (
	"sync/atomic"
	"time"
)

// CoarseTimeNow returns the current time truncated to the nearest second.
//
// This is a faster alternative to time.Now().
func CoarseTimeNow() time.Time {
	tp := coarseTime.Load().(*time.Time)
	return *tp
}

func init() {
	t := time.Now().Truncate(time.Second)
	coarseTime.Store(&t)
	go func() {
		for {
			time.Sleep(time.Second)
			t := time.Now().Truncate(time.Second)
			coarseTime.Store(&t)
		}
	}()
}

var coarseTime atomic.Value

type timer struct {
	*time.Timer
	c chan struct{}
	d time.Duration
}

func (t timer) send() {
	t.c <- struct{}{}
}
func (t timer) reset() {
	if !t.Reset(t.d) {
		if len(t.c) == 1 {
			<-t.c
		}
	}
}
func (t timer) stop() {
	if !t.Stop() {
		if len(t.c) == 1 {
			<-t.c
		}
	}
}

func newTimer(d time.Duration) timer {
	t := timer{c: make(chan struct{}), d: d}
	t.Timer = time.AfterFunc(d, t.send)
	return t
}
