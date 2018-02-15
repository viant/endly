package endly

import (
	"github.com/pkg/errors"
	"github.com/viant/toolbox/url"
)

//DockerTagRequest represents docker tag request
type DockerTagRequest struct {
	Target *url.Resource `required:"true" description:"host with docker service"` //target host
	SourceTag *DockerTag `required:"true"`
	TargetTag *DockerTag `required:"true"`
}

//DockerTag represent a docker tag
type DockerTag struct {
	Username string
	Registry string
	Image    string
	Version  string
}

//DockerTagResponse represents docker tag response
type DockerTagResponse struct {
	Stdout string
}

//Validate checks if request valid
func (r *DockerTagRequest) Validate() error {
	if r.Target == nil {
		return errors.New("target was empty")
	}
	if r.SourceTag == nil {
		return errors.New("sourceImage was empty")
	}
	if r.TargetTag == nil {
		return errors.New("sourceImage was empty")
	}
	if err := r.SourceTag.Validate(); err != nil {
		return err
	}
	return r.TargetTag.Validate()
}

//Validate checks if tag is valid
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
