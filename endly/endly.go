package main

import (
	"log"
	"os"

	"github.com/viant/endly"
	endlyclient "github.com/viant/endly/client"
	"github.com/viant/endly/server/control"
	"github.com/viant/endly/service/bootstrap"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "serve":
			if err := control.Run(os.Args[2:], endly.New); err != nil {
				log.Fatal(err)
			}
			return
		case "client":
			if err := endlyclient.Run(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			return
		}
	}
	bootstrap.Bootstrap()
}
