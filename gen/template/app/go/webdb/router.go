package webdb

import (
	"fmt"
	"github.com/viant/toolbox"
	"net/http"
)

const baseURI = "/v1/api"

type Router struct {
	*http.ServeMux
	service *Service
}

func (r Router) route() {
	r.ServeMux.Handle(baseURI+"/", r.api())
	r.ServeMux.Handle("/", r.static())
}

func (r Router) api() http.Handler {
	router := toolbox.NewServiceRouter(
		toolbox.ServiceRouting{
			HTTPMethod: "POST",
			URI:        fmt.Sprintf("%v/dummy/find", baseURI),
			Handler:    r.service.Find,
			Parameters: []string{"request"},
		},
		toolbox.ServiceRouting{
			HTTPMethod: "GET",
			URI:        fmt.Sprintf("%v/dummy/{id}", baseURI),
			Handler: func(id int) interface{} {
				return r.service.Find(&FindRequest{Id: id})
			},
			Parameters: []string{"id"},
		},

		toolbox.ServiceRouting{
			HTTPMethod: "POST",
			URI:        fmt.Sprintf("%v/dummy/get", baseURI),
			Handler:    r.service.GetAll,
			Parameters: []string{"request"},
		},
		toolbox.ServiceRouting{
			HTTPMethod: "GET",
			URI:        fmt.Sprintf("%v/dummy", baseURI),
			Handler: func() interface{} {
				return r.service.GetAll(&GetRequest{})
			},
			Parameters: []string{},
		},

		toolbox.ServiceRouting{
			HTTPMethod: "GET",
			URI:        fmt.Sprintf("%v/dummytype", baseURI),
			Handler: func() interface{} {
				return r.service.GetAllTypes(&GetTypeRequest{})
			},
			Parameters: []string{},
		},

		toolbox.ServiceRouting{
			HTTPMethod: "POST",
			URI:        fmt.Sprintf("%v/dummy", baseURI),
			Handler:    r.service.Persist,
			Parameters: []string{"request"},
		},

		toolbox.ServiceRouting{
			HTTPMethod: "POST",
			URI:        fmt.Sprintf("%v/dummy/handle", baseURI),
			Handler:    r.service.Persist,
			Parameters: []string{"request"},
		},
	)
	return http.HandlerFunc(func(writer http.ResponseWriter, reader *http.Request) {
		if err := router.Route(writer, reader); err != nil {
			http.Error(writer, err.Error(), 500)
		}

	})
}

func (r Router) static() http.Handler {
	return http.FileServer(http.Dir("static"))
}

func NewRouter(dummyService *Service) http.Handler {
	var result = &Router{
		ServeMux: http.NewServeMux(),
		service:  dummyService,
	}
	result.route()
	return result
}
