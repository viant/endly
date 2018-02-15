package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerImagesRequest represents docker check image request
type DockerImagesRequest struct {
	Target    *url.Resource `required:"true" description:"host with docker service"` //target host
	Repository string `required:"true"`
	Tag        string `required:"true"`
}

//DockerImagesResponse represents a docker check image response
type DockerImagesResponse struct {
	Images []*DockerImageInfo
}

//DockerImageInfo represents docker image info
type DockerImageInfo struct {
	Repository string
	Tag        string
	ImageID    string
	Size       int
}
