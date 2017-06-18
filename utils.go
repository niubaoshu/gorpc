package gorpc

import "strings"

func getName(fullName string) string {
	for i := len(fullName) - 1; i > 0; i-- {
		if fullName[i] == '.' {
			return strings.Title(fullName[i+1:])
		}
	}
	return ""
}
