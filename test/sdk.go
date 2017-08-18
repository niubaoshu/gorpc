package test

import "time"

var (
	pingPong func(msg string) (rmsg string, err error)
	add      func(a int) (err error)
	now      func() (time time.Time, err error)
	inc      func() (err error)
)
