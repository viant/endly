 package main

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/viant/asc"

	"flag"
	"github.com/viant/endly/example/etl/transformer"

	"log"
)

var configURI = flag.String("config", "config/config.json", "path to json config file")

func main() {
	//	flag.Parse()
	config := &transformer.Config{}
	config.Port = "8889"
	//configResource := url.NewResource(*configURI)
	//err := configResource.JSONDecode(config)
	//if err != nil {
	//		log.Fatal(err)
	//	}
	service := transformer.NewService()
	server, err := transformer.NewServer(config, service)
	if err != nil {
		log.Fatal(err)
	}
	server.Start()
}
