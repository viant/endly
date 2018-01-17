package endly

import (
	"github.com/viant/toolbox/url"
	"github.com/pkg/errors"
)

//DockerServiceTagRequest represents docker tag request
type DockerServiceTagRequest struct {
	Target    *url.Resource
	SysPath   []string
	SourceTag *DockerTag
	TargetTag *DockerTag
}

//DockerTag represent a docker tag
type DockerTag struct {
	Username string
	Registry string
	Image    string
	Version  string
}

//DockerServiceTagResponse represents docker tag response
type DockerServiceTagResponse struct {
	Stdout string
}

func (r *DockerServiceTagRequest) Validate() error {
	if r.SourceTag == nil {
		return errors.New("sourceImage was empty")
	}
	if r.TargetTag == nil {
		return errors.New("sourceImage was empty")
	}
	if err := r.SourceTag.Validate();err != nil {
		return err
	}
	if err := r.TargetTag.Validate();err != nil {
		return err
	}
	return nil
}


func (t *DockerTag) Validate() error {
	if t.Image == "" {
		return errors.New("image was empty")
	}
	return nil
}

//String stringify docker tag
func (t *DockerTag) String() string {
	var result = t.Username
	if result == "" {
		result = t.Registry
	}
	if result != "" {
		result += "/"
	}
	result += t.Image
	if t.Version != "" {
		result += ":" + t.Version
	}
	return result
}
