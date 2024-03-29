package sdk

import (
	"github.com/pkg/errors"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/system/exec"
	"strings"
)

// SetRequest represents sdk set request
type SetRequest struct {
	Sdk          string //request sdk jdk, go
	Version      string //requested version
	Env          map[string]string
	Target       *location.Resource //target host
	BaseLocation string
}

// Init initializes request
func (r *SetRequest) Init() error {
	if r.Version == "" && strings.Contains(r.Sdk, ":") {
		var fragments = strings.SplitN(r.Sdk, ":", 2)
		r.Sdk = fragments[0]
		r.Version = fragments[1]
	}
	if r.BaseLocation == "" {
		r.BaseLocation = baseLocation
	}
	r.Target = exec.GetServiceTarget(r.Target)
	return nil
}

// Validate checks if request if valid
func (r *SetRequest) Validate() error {
	if r.Sdk == "" {
		return errors.New("sdk was empty")
	}
	if r.Version == "" {
		return errors.New("version was empty")
	}
	return nil
}

// NewSetRequest creates a new sdk request
func NewSetRequest(target *location.Resource, sdk string, version string, env map[string]string) *SetRequest {
	if len(env) == 0 {
		env = make(map[string]string)
	}
	return &SetRequest{
		Target:  target,
		Sdk:     sdk,
		Version: version,
		Env:     env,
	}
}

// NewSetRequestFromURL creates a new set request from URL
func NewSetRequestFromURL(URL string) (*SetRequest, error) {
	var response = &SetRequest{}
	resource := location.NewResource(URL)
	return response, resource.Decode(response)
}

// SetResponse represents sdk response
type SetResponse struct {
	SdkInfo *Info
}

// Info represents a system sdk
type Info struct {
	Home      string //sdk path
	Build     string //sdk build version
	SessionID string //session id of target host
	Sdk       string //requested sdk
	Version   string //requested  sdk version
}
