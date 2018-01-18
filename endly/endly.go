package main

import (
	"flag"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/viant/asc"
	_ "github.com/viant/bgc"
	"github.com/viant/endly"
	_ "github.com/viant/endly/static"
	_ "github.com/viant/toolbox/storage/aws"
	_ "github.com/viant/toolbox/storage/gs"
	"log"
	"time"
	"os"
)

var workflow = flag.String("r", "run.json", "path to workflow run request json file")

func main() {
	flag.Parse()
	runner := endly.NewCliRunner()
	var arguments = make([]interface{}, 0)
	if len(os.Args) > 1 {
		for i := 1;i<len(os.Args);i++{
			if os.Args[i] == "-r" {
				i++
				continue
			}
			arguments = append(arguments, os.Args[i])
		}
	}
	err := runner.Run(*workflow, arguments...)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
}
