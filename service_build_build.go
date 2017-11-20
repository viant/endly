package endly

import (
	"github.com/viant/toolbox/url"
	"fmt"
	"github.com/pkg/errors"
)

//BuildSpec represents build specification.
type BuildSpec struct {
	Name      string //build system name, i.e go, mvn, node, yarn
	Version   string //build system version
	Goal      string //lookup for BuildMeta goal
	BuildGoal string //actual build target, like clean, test
	Args      string // additional build arguments , that can be expanded with $build.args
	Sdk        string
	SdkVersion string
}

//BuildRequest represents a build request.
type BuildRequest struct {
	MetaURL string
	BuildSpec    *BuildSpec    //build specification
	Target       *url.Resource //path to application to be build, Note that command may use $build.target variable. that expands to Target URL path
}


//BuildResponse represents a build response.
type BuildResponse struct {
	CommandInfo *CommandResponse
}

//Validate validates if request is valid
func (r *BuildRequest) Validate() error {
	if r.BuildSpec == nil {
		return errors.New("BuildSpec was empty")
	}
	if r.BuildSpec.Name == "" {
		return fmt.Errorf("BuildSpec.Name was empty for %v", r.BuildSpec.Name)
	}
	if r.BuildSpec.Goal == "" {
		return fmt.Errorf("BuildSpec.Goal was empty for %v", r.BuildSpec.Name)
	}
	return nil
}