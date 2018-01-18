package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerInspectRequest represents a docker inspect request, target name refers to container name
type DockerInspectRequest struct {
	Target *url.Resource
}

//DockerInspectResponse represents a docker inspect request
type DockerInspectResponse struct {
	Output string
	Info   interface{} //you can extract any instance default, for instance to get Ip you can use Output[0].NetworkSettings.IPAddress in the variable action post from key
}
