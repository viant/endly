package cloudscheduler

import (
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"google.golang.org/api/cloudscheduler/v1beta1"
	"strings"
)

//DeployJobRequest represents deploy job request
type DeployRequest struct {
	cloudscheduler.Job
	Region  string
	Body    string
	jobName string
	parent  string
	project string
}

func (r *DeployRequest) Validate() error {
	if r.Name == "" {
		return errors.Errorf("name was empty")
	}
	return nil
}

func (r *DeployRequest) Parent(context *endly.Context) string {
	if r.parent == "" {
		r.parent = "projects/${gcp.projectId}/locations/${gcp.region}"
	}
	return gcp.ExpandMeta(context, r.parent)
}

func (r *DeployRequest) Init() error {
	if r.Name == "" {
		return nil
	}
	elements := strings.Split(r.Name, "/")
	if len(elements) == 1 {
		r.jobName = r.Name
		r.parent = "projects/${gcp.projectId}/locations/${gcp.region}"

	} else if len(elements) > 4 {
		r.jobName = elements[len(elements)-1]
		r.parent = strings.Join(elements[:4], "/")
	}
	return nil
}

//DeployJobResponse represents deploy job response
type DeployResponse struct {
	*cloudscheduler.Job
}
