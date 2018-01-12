package main

import (
	"flag"
	"log"
	"github.com/viant/toolbox/url"
	"github.com/viant/endly/example/rt/elogger"
)

var configURI = flag.String("config", "", "path to json config file")

func main() {
	flag.Parse()
	config := &elogger.Config{}

	configResource := url.NewResource(*configURI)
	err := configResource.JSONDecode(config)
	if err != nil {
		log.Fatal(err)
	}

	service, err := elogger.NewService(config)
	if err != nil {
		log.Fatal(err)
	}
	server, err := elogger.NewServer(config, service)
	if err != nil {
		log.Fatal(err)
	}
	server.Start()
}
