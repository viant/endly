package main

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/viant/asc"

	"flag"
	"github.com/viant/endly/example/etl/transformer"
	"log"
	"github.com/viant/toolbox/url"
)

var configURI = flag.String("config", "", "path to json config file")

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
