package main

import (
	_ "github.com/adrianwit/mgc"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/viant/asc"
	_ "github.com/viant/bgc"

	.  ".."
	"log"
	"os"
	"fmt"
	"flag"
)


var configURL = flag.String("configURL", "", "path to config file (JSON or YAML")

func main() {
	flag.Parse()
	config, err := NewConfigFromURL(*configURL)
	if err != nil {
		log.Fatal(err)
	}
	service, err := New(config.Datastore)
	if err != nil {
		log.Fatal(err)
	}
	server := NewServer(service, config.Port)
	go server.StopOnSiginals(os.Interrupt)
	fmt.Printf("start listening on :%d\n", config.Port)
	server.ListenAndServe()

}
