package gotiny

import (
	"unsafe"
)

func floatToUint(f float64) uint64 {
	return reverseByte(*((*uint64)(unsafe.Pointer(&f))))
}

func uintToFloat(u uint64) float64 {
	u = reverseByte(u)
	return *((*float64)(unsafe.Pointer(&u)))
}

func reverseByte(u uint64) uint64 {
	u = (u << 32) | (u >> 32)
	u = ((u << 16) & 0xFFFF0000FFFF0000) | ((u >> 16) & 0xFFFF0000FFFF)
	u = ((u << 8) & 0xFF00FF00FF00FF00) | ((u >> 8) & 0xFF00FF00FF00FF)
	return u
}

//int -5 -4 -3 -2 -1 0 1 2 3 4 5 6
//uint 9  7  5  3  1 0 2 4 6 8 10 12
func intToUint(v int64) uint64 {
	return uint64((v << 1) ^ (v >> 63))
}

//uint 9  7  5  3  1 0 2 4 6 8 10 12
//int -5 -4 -3 -2 -1 0 1 2 3 4 5 6
func uintToInt(u uint64) int64 {
	v := int64(u)
	return (-(v & 1)) ^ (v>>1)&0x7FFFFFFFFFFFFFFF
	//return (-(v & 1)) ^ (v>>1)&0x7FFFFFFFFF
	//return (-(v & 1)) ^ (v >> 1)
}
