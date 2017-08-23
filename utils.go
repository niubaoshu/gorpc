package gorpc

import (
	"reflect"
	"runtime"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

func getNameByPtr(ptr uintptr) string {
	fullName := getFullNameByPtr(ptr)
	for i := len(fullName) - 1; i > 0; i-- {
		if fullName[i] == '.' {
			return fullName[i+1:]
		}
	}
	return fullName
}

func getNameByFunc(f interface{}) string {
	return getNameByPtr(reflect.ValueOf(f).Pointer())
}

func getFullNameByFunc(f interface{}) string {
	return getFullNameByPtr(reflect.ValueOf(f).Pointer())
}
func getFullNameByPtr(ptr uintptr) string {
	return runtime.FuncForPC(ptr).Name()
}

func firstToUpper(str string) string {
	r, n := utf8.DecodeRuneInString(str)
	if unicode.IsLower(r) {
		ur := unicode.ToUpper(r)
		l := utf8.RuneLen(ur)
		buf := make([]byte, l)
		utf8.EncodeRune(buf, ur)
		return string(buf) + str[n:]
	}
	return str
}

type vals struct {
	rvs  []reflect.Value
	ptrs []unsafe.Pointer
}

func max(list []int) int {
	m := 0
	l := len(list)
	for i := 0; i < l; i++ {
		if list[i] > m {
			m = list[i]
		}
	}
	return m
}
