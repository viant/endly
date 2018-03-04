package build

import (
	"errors"
	"fmt"
	"github.com/viant/endly/deployment/deploy"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/system/storage"
	"github.com/viant/toolbox/url"
)

//Spec represents build specification.
type Spec struct {
	Name       string `required:"true" description:"build system name, i.e go, mvn, node, yarn, build system meta is defined in meta/build/XXX"`
	Version    string `required:"true" description:"build system version"`
	Goal       string `required:"true" description:"build goal to be matched with build meta goal"`
	BuildGoal  string `required:"true" description:"actual build target, like clean, test"`
	Args       string `required:"true" description:"additional build arguments , that can be expanded with $build.args in build meta"`
	Sdk        string
	SdkVersion string
}

//Request represents a build request.
type Request struct {
	MetaURL     string            `description:"build meta URL"`
	BuildSpec   *Spec             `required:"true" description:"build specification" `
	Credentials map[string]string `description:"key value pair of placeholder and credential files, check build meta file for used placeholders i.e for 'go' build: ##git## - git usernamem, **git** - git password"`
	Env         map[string]string `description:"environmental variables"`
	Target      *url.Resource     `required:"true" description:"build location, host and path" `
}

//Response represents a build response.
type Response struct {
	CommandInfo *exec.CommandResponse
}

//Validate validates if request is valid
func (r *Request) Validate() error {
	if r.BuildSpec == nil {
		return errors.New("buildSpec was empty")
	}
	if r.BuildSpec.Name == "" {
		return fmt.Errorf("buildSpec.Name was empty for %v", r.BuildSpec.Name)
	}
	if r.BuildSpec.Goal == "" {
		return fmt.Errorf("buildSpec.Goal was empty for %v", r.BuildSpec.Name)
	}
	return nil
}

//LoadMetaRequest represents a loading Meta request
type LoadMetaRequest struct {
	Source *url.Resource `required:"true" description:"URL with build meta JSON"`
}

//Validate checks if request is valid
func (r *LoadMetaRequest) Validate() error {
	if r.Source == nil {
		return errors.New("source was empty")
	}
	return nil
}

//LoadMetaResponse represents build meta response.
type LoadMetaResponse struct {
	Meta *Meta //url to size
}

//Goal builds goal represents a build goal
type Goal struct {
	Name                string                   `required:"true"`
	InitTransfers       *storage.CopyRequest     ` description:"files transfer before build command"`
	Command             *exec.ExtractableCommand `required:"true"  description:"build command"`
	PostTransfers       *storage.CopyRequest     ` description:"files transfer after build command"`
	VerificationCommand *exec.ExtractableCommand
}

//Meta build meta provides instruction how to build an app
type Meta struct {
	Name         string               `required:"true" description:"name of build system"`
	Goals        []*Goal              `required:"true" description:"build goals"`
	Dependencies []*deploy.Dependency `description:"deployment dependencies"`
	goalsIndex   map[string]*Goal
}

//Validate validates build meta.
func (m *Meta) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("metaBuild.Names %v", m.Name)

	}
	if len(m.Goals) == 0 {
		return fmt.Errorf("metaBuild.Goals were empty %v", m.Name)
	}
	return nil
}
