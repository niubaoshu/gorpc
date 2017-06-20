package gorpc

import (
	"reflect"
	"runtime"
	"strings"
)

func getNameByPtr(ptr uintptr) string {
	fullName := getFullNameByPtr(ptr)
	for i := len(fullName) - 1; i > 0; i-- {
		if fullName[i] == '.' {
			return strings.Title(fullName[i+1:])
		}
	}
	return ""
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
