package endly

import (
	"github.com/pkg/errors"
	"github.com/viant/toolbox/url"
)

//DockerBuildRequest represents docker build request
type DockerBuildRequest struct {
	Target    *url.Resource `required:"true" description:"host with docker service"` //target host
	Tag       *DockerTag    `required:"true" description:"build docker tag"`
	Path      string        `required:"true" description:"docker build source path"`
	Arguments map[string]string `description:"docker build command line arguments, see more: https://docs.docker.com/engine/reference/commandline/build/#description "` //https://docs.docker.com/engine/reference/commandline/build/#description
}

//DockerBuildResponse represents docker build response
type DockerBuildResponse struct {
	Stdout string
}

//Init initialises default values
func (r *DockerBuildRequest) Init() {
	if len(r.Arguments) == 0 && r.Tag != nil {
		r.Arguments = make(map[string]string)
	}
	if r.Tag != nil {
		r.Arguments["-t"] = r.Tag.String()
	}
}

//Validate check if request is valid
func (r *DockerBuildRequest) Validate() error {
	if r.Target == nil {
		return errors.New("target was nil")
	}
	if r.Path == "" {
		return errors.New("path was empty was nil")
	}
	if r.Tag != nil {
		err := r.Tag.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}
