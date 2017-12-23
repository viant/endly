package main

import (
	"flag"
	"github.com/viant/endly/example/etl/transformer"
	"github.com/viant/toolbox/url"
	"log"
)

var configURI = flag.String("config", "config/config.json", "path to json config file")

func main() {
	flag.Parse()
	config := &transformer.Config{}
	configResource := url.NewResource(*configURI)
	err := configResource.JSONDecode(config)
	if err != nil {
		log.Fatal(err)
	}
	service := transformer.NewService()
	server, err := transformer.NewServer(config, service)
	if err != nil {
		log.Fatal(err)
	}
	server.Start()
}
