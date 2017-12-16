package endly

import "github.com/viant/toolbox/url"

//SeleniumServerStopRequest represents server stop request
type SeleniumServerStopRequest struct {
	Target *url.Resource
	Port   int
}

//SeleniumServerStopResponse represents a selenium stop request
type SeleniumServerStopResponse struct {
}
