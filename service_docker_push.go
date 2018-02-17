package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerPushRequest represents a docker push request
type DockerPushRequest struct {
	Target *url.Resource `required:"true" description:"host with docker service"` //target host
	Tag    *DockerTag    `required:"true"`
}

//DockerPushResponse represents a docker push request
type DockerPushResponse struct {
}
