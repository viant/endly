package main

import (
	"fmt"
	"github.com/viant/endly"
	_ "github.com/viant/endly/gen/static"
	"github.com/viant/endly/gen/web"
	"github.com/viant/toolbox"
	"log"
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

	router := web.NewRouter(service, func(request *http.Request) {})

	server := &http.Server{Addr: ":8071", Handler: router}
	server.SetKeepAlivesEnabled(false)
	log.Fatal(server.ListenAndServe())

}
