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


	msbt := NewMSBT("./nh/TalkNNpc_USen/B1_Bo/Free/BO_FreeA_Always.msbt")

	writeMSBT(msbt, "./mousebt.msbt")

	
	for _,lbl := range msbt.Lbl1.Labels {
		fmt.Println(lbl.Index)
		fmt.Println(string(lbl.Value))
	}
	//fmt.Printf("%+v", NewMSBT("./nh/TalkNNpc_USen/B1_Bo/Free/BO_FreeA_Always.msbt").Header)
	//fmt.Printf("%+v", NewMSBT("./nh/TalkNNpc_USen/B1_Bo/Free/BO_FreeA_Always.msbt"))
}
