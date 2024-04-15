package main

import (
	"bytes"
	"io"
	"os"
)


func main() {
	file,err := os.Create("./converted/out.msbt")
	if err != nil {
		panic("Could not create file")
	}
	buf := &bytes.Buffer{}
	_,err = buf.Write([]byte("MsgStdBn"))
	if err != nil {
		panic("Could not write buffer")
	}
	io.Copy(file,buf)
	println("Completed.")
}
