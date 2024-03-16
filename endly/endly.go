package main

import (
	"github.com/viant/endly/bootstrap"
	"os"
)

func main() {
	os.Chdir("/Users/awitas/go/src/github.com/viant/bqtail/e2e")
	bootstrap.Bootstrap()
}
