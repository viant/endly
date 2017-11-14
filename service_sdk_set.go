package endly

import "github.com/viant/toolbox/url"

//SystemSdkSetRequest represents sdk set request
type SystemSdkSetRequest struct {
	Sdk     string //request sdk jdk, go
	Version string //requested version
	Env     map[string]string
	Target  *url.Resource //target host
}

//SystemSdkSetResponse represents sdk response
type SystemSdkSetResponse struct {
	Home      string //sdk path
	Build     string //sdk build version
	SessionID string //session id of target host
	Sdk       string //requested sdk
	Version   string //requested  sdk version
}
