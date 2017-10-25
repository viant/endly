package endly

import "github.com/viant/toolbox/url"

//BuildSpec represents build specification.
type BuildSpec struct {
	Name       string //build Id  like go, mvn, node, yarn
	Version    string
	Goal       string //lookup for BuildMeta goal
	BuildGoal  string //actual build target, like clean, test
	Args       string // additional build arguments , that can be expanded with $build.args
	Sdk        string
	SdkVersion string
}

//BuildRequest represents a build request.
type BuildRequest struct {
	BuildMetaURL string
	BuildSpec    *BuildSpec    //build specification
	Target       *url.Resource //path to application to be build, Note that command may use $build.target variable. that expands to Target URL path
}

//BuildResponse represents a build response.
type BuildResponse struct {
	SdkResponse *SystemSdkSetResponse
	CommandInfo *CommandResponse
}
