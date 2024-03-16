package main

import (
	"flag"
	_ "github.com/viant/asc"
	"github.com/viant/endly/example/ui/sso"
	"github.com/viant/endly/model/location"
	"log"
)

var configURI = flag.String("config", "config/config.json", "path to json config file")

func main() {
	flag.Parse()
	config := &sso.Config{}
	configResource := location.NewResource(*configURI)
	err := configResource.Decode(config)
	if err != nil {
		log.Fatal(err)
	}

	service, err := sso.NewService(config)
	if err != nil {
		log.Fatal(err)
	}
	server, err := sso.NewServer(config, service)
	if err != nil {
		log.Fatal(err)
	}
	server.Start()
}
