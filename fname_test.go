package gorpc

import (
	"testing"
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
	if getNameByFunc(upper) != "upper" {
		t.Error("getNameByFunc upper", getNameByFunc(upper))
	}
}
