package endly

import (
	"github.com/viant/toolbox/url"
	"github.com/pkg/errors"
)

//DockerBuildRequest represents docker build request
type DockerBuildRequest struct {
	Target    *url.Resource
	SysPath   []string
	Tag       *DockerTag
	Path      string
	Arguments map[string]string //https://docs.docker.com/engine/reference/commandline/build/#description
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


//Vaidate check if request is valid
func (r *DockerBuildRequest) Vaidate() error {
	if r.Target == nil {
		return errors.New("target was nil")
	}
	if r.Tag != nil {
		err := r.Tag.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}
