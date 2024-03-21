package main

import (
	"github.com/viant/endly/bootstrap"
	"os"
)

func main() {
	os.Chdir("/Users/awitas/go/src/github.com/viant/endly/model/transformer/transfer/testdata/projectx")
	//os.Args = []string{"endly", "-r=regression/regression", "-p", "-f","yaml"}
	bootstrap.Bootstrap()
}
