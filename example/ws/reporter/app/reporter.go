package main

import (
	"flag"
	_ "github.com/go-sql-driver/mysql"
	"github.com/viant/endly/example/ws/reporter"
	"github.com/viant/endly/model/location"
	"log"
)

var configURI = flag.String("config", "config/config.json", "path to json config  file")
var port = flag.String("port", "8085", "service port")

func main() {
	flag.Parse()
	config := &reporter.Config{}
	configResource := location.NewResource(*configURI)
	err := configResource.Decode(config)
	if err != nil {
		log.Fatal(err)
	}
	service, err := reporter.NewService(config)
	if err != nil {
		log.Fatal(err)
	}
	server := reporter.NewServer(*port, service)
	if err != nil {
		log.Fatal(err)
	}
	server.Start()
}
