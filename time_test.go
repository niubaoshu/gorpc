package gorpc

//
//import (
//	"testing"
//	"time"
//)
//
//func BenchmarkTime(b *testing.B) {
//	t := newTimer(time.Nanosecond)
//	for i := 0; i < b.N; i++ {
//		t.start()
//		<-t.c
//		t.stop()
//	}
//}
//
//func BenchmarkTimer(b *testing.B) {
//	t := time.NewTimer(time.Nanosecond)
//	for i := 0; i < b.N; i++ {
//		t.Reset(time.Nanosecond)
//		a := <-t.C
//		_ = a.Second()
//		t.Stop()
//	}
//}
