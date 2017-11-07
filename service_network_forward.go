package endly

import "github.com/viant/toolbox/url"

//NetworkForward represents network link
type NetworkForward struct {
	Local string
	Remote string
}


//NetworkRequest represents a net forwarding request
type NetworkForwardRequest struct {
	Target        *url.Resource
	Forwards []*NetworkForward
}

//NetworkForwardResponse represents expanded net forward rule
type NetworkForwardResponse struct {
	Forwards []*NetworkForward
}
