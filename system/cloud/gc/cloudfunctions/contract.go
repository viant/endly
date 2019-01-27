package cloudfunctions

import (
	"github.com/viant/toolbox/url"
	"google.golang.org/api/cloudfunctions/v1"
)

type DeployRequest struct {
	*cloudfunctions.CloudFunction
	Source      *url.Resource
	Location    string
	Credentials string
}
