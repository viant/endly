package sdk

import "github.com/viant/toolbox/url"

//SetRequest represents sdk set request
type SetRequest struct {
	Sdk     string //request sdk jdk, go
	Version string //requested version
	Env     map[string]string
	Target  *url.Resource //target host
}

//SetResponse represents sdk response
type SetResponse struct {
	SdkInfo *Info
}

//Info represents a system sdk
type Info struct {
	Home      string //sdk path
	Build     string //sdk build version
	SessionID string //session id of target host
	Sdk       string //requested sdk
	Version   string //requested  sdk version
}
