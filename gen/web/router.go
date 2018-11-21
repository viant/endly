package web

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

const baseURI = "/v1/api"

type Router struct {
	mem      storage.Service
	service  *Service
	callback func(request *http.Request)
}

func (r *Router) route() {

	http.Handle(baseURI+"/", r.api())
	http.Handle("/", r.static())
	http.Handle("/download/", r.download())

}

func (r *Router) api() http.Handler {
	router := toolbox.NewServiceRouter(
		toolbox.ServiceRouting{
			HTTPMethod: "GET",
			URI:        fmt.Sprintf("%v/meta", baseURI),
			Handler: func() interface{} {
				resp, err := r.service.Get(&GetRequest{})
				if err != nil {
					resp = &GetResponse{
						Status: "error",
						Error:  err.Error(),
					}
				}
				return resp
			},
			Parameters: []string{},
		},
	)
	return http.HandlerFunc(func(writer http.ResponseWriter, reader *http.Request) {
		if err := router.Route(writer, reader); err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}

	})
}

func (r *Router) setTextValue(form url.Values, field string, target *string, defaultValue string) {
	if value, ok := form[field]; ok {
		textValue := value[0]
		textValue = strings.Replace(textValue, " ", "", len(textValue))
		*target = textValue
	} else {
		*target = defaultValue
	}
}

func (r *Router) setBoolValue(form url.Values, field string, target *bool) {
	if value, ok := form[field]; ok {
		boolValue := value[0] != ""
		*target = boolValue
	}
}

func (r *Router) download() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if r.callback != nil {
			r.callback(request)
		}
		request.ParseForm()
		var form = request.Form
		var runRequest = &RunRequest{
			Datastore: []*Datastore{},
			Build:     &Build{},
			Testing:   &Testing{},
		}
		r.setTextValue(form, "appTemplate", &runRequest.Build.TemplateApp, "default")
		r.setTextValue(form, "appName", &runRequest.Build.App, "myapp")
		r.setTextValue(form, "origin", &runRequest.Origin, "")

		r.setTextValue(form, "sdk", &runRequest.Build.Sdk, "")
		r.setBoolValue(form, "docker", &runRequest.Build.Docker)

		if len(runRequest.Build.App) < 3 {
			http.Error(writer, "app name too short", 500)
			return
		}

		dbConfig := false
		r.setBoolValue(form, "dbConfig", &dbConfig)
		dbName := ""
		r.setTextValue(form, "dbName", &dbName, "db")
		dbEngines := form["dbEngine"]
		dbNames := strings.Split(dbName, ",")

		if len(dbNames) == 0 {
			dbNames = append(dbNames, "mysql")
		}
		for i := len(dbNames); i < len(dbEngines); i++ {
			dbNames = append(dbNames, fmt.Sprintf("db%v", i+1))
		}
		for i, dbName := range dbNames {
			dbNames[i] = strings.Replace(dbName, " ", "", len(dbName))
		}
		runRequest.Datastore = []*Datastore{}
		for i, driver := range dbEngines {
			if driver == "" {
				continue
			}
			runRequest.Datastore = append(runRequest.Datastore,
				&Datastore{
					Driver: driver,
					Name:   dbNames[i],
					Config: dbConfig,
				})
		}
		r.setTextValue(form, "regression", &runRequest.Testing.Regression, "inline")
		r.setBoolValue(form, "http", &runRequest.Testing.HTTP)
		r.setBoolValue(form, "rest", &runRequest.Testing.REST)
		r.setBoolValue(form, "selenium", &runRequest.Testing.Selenium)
		r.setBoolValue(form, "inline", &runRequest.Testing.Inline)
		r.setBoolValue(form, "ssh", &runRequest.Testing.SSH)
		r.setBoolValue(form, "dataValidation", &runRequest.Testing.DataValidation)
		r.setBoolValue(form, "logValidation", &runRequest.Testing.LogValidation)
		r.setTextValue(form, "useCaseData", &runRequest.Testing.UseCaseData, "")

		if len(runRequest.Datastore) > 0 {
			r.setBoolValue(form, "mapping", &runRequest.Datastore[0].MultiTableMapping)
		}
		resp, err := r.service.Run(runRequest)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/zip")
		writer.Write(resp.Data)
	})
}

func (r *Router) static() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var assetPath = string(request.URL.Path[1:])
		if assetPath == "" {
			assetPath = "index.html"
		}
		var URL = toolbox.URLPathJoin(r.service.baseAssetURL, assetPath)
		if has, _ := r.mem.Exists(URL); !has {
			http.NotFound(writer, request)
			return
		}

		var ext = path.Ext(assetPath)
		if ext != "" {
			ext = string(ext[1:])
		}
		contentType, has := toolbox.FileExtensionMimeType[ext]
		if !has {
			contentType = fmt.Sprintf("text/%v", ext)
		}
		if strings.Contains(contentType, "text") {
			contentType += "; charset=utf-8"
		}
		writer.Header().Set("Content-Type", contentType)
		object, err := r.mem.StorageObject(URL)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		reader, err := r.mem.Download(object)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		io.Copy(writer, reader)
	})
}

func NewRouter(service *Service, callback func(request *http.Request)) *Router {
	srv := storage.NewMemoryService()
	var result = &Router{
		service:  service,
		mem:      srv,
		callback: callback,
	}
	result.route()
	return result
}
