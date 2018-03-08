package sdk

import (
	"github.com/viant/toolbox/url"
	"strings"
	"github.com/pkg/errors"
)

//SetRequest represents sdk set request
type SetRequest struct {
	Sdk     string //request sdk jdk, go
	Version string //requested version
	Env     map[string]string
	Target  *url.Resource //target host
}


//Init initializes request
func (r *SetRequest) Init() error {
	if r.Version == "" && strings.Contains(r.Sdk , ":") {
		var fragments = strings.SplitN(r.Sdk, ":",2)
		r.Sdk = fragments[0]
		r.Version = fragments[1]
	}
	return nil
}


//Validate checks if request if valid
func (r *SetRequest) Validate() error {
	if r.Sdk == ""  {
		return errors.New("sdk was empty")
	}
	if r.Version == ""  {
		return errors.New("version was empty")
	}
	return nil
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
