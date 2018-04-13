package main

import (
	"github.com/viant/endly/gen/web"
	_ "github.com/viant/endly/gen/static"
	"net/http"
	"github.com/viant/toolbox"
	"github.com/viant/endly"
	"fmt"
)

func main() {

	//caller := toolbox.CallerDirectory(3)
	//parent := strings.Replace(caller, "app/", "", 1)
	//var baseURL = url.NewResource(parent).URL

	baseURL := fmt.Sprintf("mem://%v", endly.Namespace)
	service := web.NewService(
		toolbox.URLPathJoin(baseURL , "template"),
		toolbox.URLPathJoin(baseURL , "asset"),
	)

	web.NewRouter(service)

	http.ListenAndServe(":8080", nil)
}
