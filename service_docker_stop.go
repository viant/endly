package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerStopImagesRequest represents docker stop running images request
type DockerStopImagesRequest struct {
	Target *url.Resource `required:"true" description:"host with docker service"` //target host
	Images []string `required:"true"`
}

//DockerStopImagesResponse represents docker stop images response
type DockerStopImagesResponse struct {
	StoppedImages []string
}
