package endly

import "github.com/viant/toolbox/url"

//SeleniumServerStartRequest represents a selenium server start request
type SeleniumServerStartRequest struct {
	Target     *url.Resource
	Port       int
	Sdk        string
	SdkVersion string
	Version    string
}

//SeleniumServerStartResponse repreents a selenium server stop request
type SeleniumServerStartResponse struct {
	Pid                int
	SeleniumServerPath string
	GeckodriverPath    string
}
