package main

import (
	"github.com/viant/endly/bootstrap"
	"os"
)


func main() {
	os.Chdir("/Users/awitas/go/src/github.com/viant/datly/e2e/local")
	bootstrap.Bootstrap()
}
