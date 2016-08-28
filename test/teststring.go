package main

import (
	"fmt"
	"github.com/niubaoshu/gorpc/gotiny"
)

func main() {
	var b = []byte{
		1, 2,
	}
	var eb []byte
	bp := &b
	bpp := &bp
	e := gotiny.NewEncoder(eb, 0)
	b0p := &b[0]
	e.Encodes(&b0p)
	e.Encodes(&bpp)
	c := e.Bytes()
	d := gotiny.NewDecoder(c)
	fmt.Println(c)
	var db []byte
	var aaaa byte
	bbbb := &aaaa
	cccc := &bbbb
	dbp := &db
	d.Decodes(&cccc)
	fmt.Println(aaaa)
	fmt.Println(*bbbb)
	fmt.Println(**cccc)
	d.Decodes(&dbp)
	fmt.Println(db)
}
