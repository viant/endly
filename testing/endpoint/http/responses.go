package http

import "github.com/viant/toolbox/bridge"

//HTTPResponses represents HTTPResponses
type HTTPResponses struct {
	Request   *bridge.HttpRequest
	Responses []*bridge.HttpResponse
	Index     uint32
}

