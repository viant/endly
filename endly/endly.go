package main

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	_ "github.com/viant/asc"
	_ "github.com/viant/bgc"
	_ "github.com/viant/endly/static" //load external resource like .csv .json files to mem storage
	_ "github.com/viant/toolbox/storage/aws"
	_ "github.com/viant/toolbox/storage/gs"

	"github.com/viant/endly/bootstrap"
)

func main() {
	bootstrap.Bootstrap()
}
