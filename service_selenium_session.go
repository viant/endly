package endly

import (
	"github.com/viant/toolbox/url"
	"errors"
)


//SeleniumOpenSessionRequest represents open session request
type SeleniumOpenSessionRequest struct {
	Browser        string
	RemoteSelenium *url.Resource //remote selenium resource
}



//Validate validate open session request
func (r *SeleniumOpenSessionRequest) Validate() error {
	if r.RemoteSelenium == nil {
		return errors.New("Remote (remote selenium endpoint) was empty")
	}
	if r.RemoteSelenium.URL == "" {
		return errors.New("Remote.URL (selenium resource URL) was empty")
	}
	if r.Browser == "" {
		return errors.New("Browser was empty")
	}
	return nil
}

//SeleniumOpenSessionResponse represents open session response.
type SeleniumOpenSessionResponse struct {
	SessionID string
}

