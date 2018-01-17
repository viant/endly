package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerBuildRequest represents docker build request
type DockerBuildRequest struct {
	Target  *url.Resource
	SysPath []string
	Arguments map[string]string //https://docs.docker.com/engine/reference/commandline/build/#description
}



//DockerBuildResponse represents docker build response
type DockerBuildResponse struct {
	Stdout string
}
