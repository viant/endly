package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerRunRequest represents a docker run request
type DockerRunRequest struct {
	SysPath    []string
	Target     *url.Resource
	Image      string
	Port       string
	Credential string
	Env        map[string]string
	Mount      map[string]string
	MappedPort map[string]string
	Params     map[string]string
	Workdir    string
}
