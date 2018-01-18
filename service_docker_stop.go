package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerStopImagesRequest represents docker stop running images request
type DockerStopImagesRequest struct {
	Target *url.Resource
	Images []string
}

//DockerStopImagesResponse represents docker stop images response
type DockerStopImagesResponse struct {
	StoppedImages []string
}
