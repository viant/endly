package webdb

import (
	"net/http"
)

type Response struct {
	Status     string
	Error      string
	StatusCode *int `json:"-"`
}

func NewResponse() *Response {
	var response = &Response{Status: "ok"}
	return response
}

func (r *Response) SetError(err error) {
	r.Status = "error"
	r.Error = err.Error()
}

func (r *Response) GetStatusCode() int {
	if r.StatusCode != nil {
		return *r.StatusCode
	}
	return http.StatusOK
}

type FindRequest struct {
	Id int
}

type FindResponse struct {
	*Response
	Data *Dummy
}

type GetRequest struct {
	Id int
}

type GetResponse struct {
	*Response
	Data []*Dummy
}


type GetTypeRequest struct {
	Id int
}

type GetTypeResponse struct {
	*Response
	Data []*DummyType
}

type PersistRequest struct {
	Data *Dummy
}

type PersistResponse struct {
	*Response
	Data *Dummy
}
