package main

import (
	//"fmt"
	//"github.com/viant/endly"
	"fmt"
	"github.com/viant/endly"
	_ "github.com/viant/endly/gen/static"
	"github.com/viant/endly/gen/web"
	"github.com/viant/toolbox"
	"net/http"
)

func main() {

	//caller := toolbox.CallerDirectory(3)
	//parent := strings.Replace(caller, "app/", "", 1)
	//var baseURL = url.NewResource(parent).URL

	baseURL := fmt.Sprintf("mem://%v", endly.Namespace)
	service := web.NewService(
		toolbox.URLPathJoin(baseURL, "template"),
		toolbox.URLPathJoin(baseURL, "asset"),
	)
	web.NewRouter(service, func(request *http.Request) {})
	http.ListenAndServe(":8071", nil)

}
