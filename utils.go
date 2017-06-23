package gorpc

import (
	"reflect"
	"runtime"
	"unicode"
	"unicode/utf8"
)

func getNameByPtr(ptr uintptr) string {
	fullName := getFullNameByPtr(ptr)
	for i := len(fullName) - 1; i > 0; i-- {
		if fullName[i] == '.' {
			return firstToUpper(fullName[i+1:])
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
