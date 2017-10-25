package endly

import "github.com/viant/toolbox/url"

//ProcessStopRequest represents a stop request
type ProcessStopRequest struct {
	Target *url.Resource
	Pid    int
}
