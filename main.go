package main

import (
	"bytes"
	"fmt"
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

	fmt.Printf("%+v", NewMSBT("./nh/TalkNNpc_USen/B1_Bo/Free/BO_FreeA_Always.msbt").Header)
	fmt.Printf("%+v", NewMSBT("./nh/TalkNNpc_USen/B1_Bo/Free/BO_FreeA_Always.msbt"))
}
