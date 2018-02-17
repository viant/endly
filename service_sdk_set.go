package endly

import "github.com/viant/toolbox/url"

//SdkSetRequest represents sdk set request
type SdkSetRequest struct {
	Sdk     string //request sdk jdk, go
	Version string //requested version
	Env     map[string]string
	Target  *url.Resource //target host
}

//SdkSetResponse represents sdk response
type SdkSetResponse struct {
	SdkInfo *SystemSdkInfo
}

//SystemSdkInfo represents a system sdk
type SystemSdkInfo struct {
	Home      string //sdk path
	Build     string //sdk build version
	SessionID string //session id of target host
	Sdk       string //requested sdk
	Version   string //requested  sdk version
}
