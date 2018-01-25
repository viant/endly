package endly

import (
	"fmt"
	"github.com/viant/toolbox/url"
)

//DockerRunRequest represents a docker run request
type DockerRunRequest struct {
	Target      *url.Resource
	Image       string
	Port        string
	Credentials map[string]string //container credential, i,e mysql password
	Env         map[string]string
	Mount       map[string]string
	MappedPort  map[string]string
	Params      map[string]string
	Workdir     string
}

//Validate checks if request is valid
func (r *DockerRunRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("target was nil")
	}
	if r.Target.Name == "" {
		return fmt.Errorf("target name was empty for %v", r.Target.URL)
	}
	if r.Image == "" {
		return fmt.Errorf("image was empty for %v", r.Target.URL)
	}
	return nil
}
