package main

import (
	"flag"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/viant/asc"
	_ "github.com/viant/bgc"
	"github.com/viant/endly"
	_ "github.com/viant/toolbox/storage/aws"
	_ "github.com/viant/toolbox/storage/gs"
	"log"
	"time"
)

var workflow = flag.String("workflow", "run.json", "path to workflow run request json file")

func main() {
	flag.Parse()
	runner := endly.NewCliRunner()
	err := runner.Run(*workflow)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
}
