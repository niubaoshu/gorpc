package gorpc

import (
	"strings"
	"testing"
	"time"
)

func upper() {}

func TestFirstLetterToLower(t *testing.T) {
	src := []string{"asdf", "aAAA", "Asdfsfsfds"}
	dst := []string{"Asdf", "AAAA", "Asdfsfsfds"}
	for i := 0; i < len(src); i++ {
		if dst[i] != firstToUpper(src[i]) {
			t.Fatalf("firstToLower err %s", firstToUpper(src[i]))
		}
	}
}

func TestGetNameByFunc(t *testing.T) {
	src := []interface{}{upper, time.Now, time.LoadLocation, strings.ToUpper, TestGetNameByFunc}
	des := []string{"upper", "Now", "LoadLocation", "ToUpper", "TestGetNameByFunc"}
	for i := 0; i < len(src); i++ {
		if getNameByFunc(src[i]) != des[i] {
			t.Errorf("getNameByFunc %s", getNameByFunc(upper))
		}
	}
}
