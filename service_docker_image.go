package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerImagesRequest represents docker check image request
type DockerImagesRequest struct {
	SysPath    []string
	Target     *url.Resource
	Repository string
	Tag        string
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
