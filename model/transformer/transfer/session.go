package transfer

import (
	"github.com/viant/afs/file"
	"github.com/viant/afs/url"
	"github.com/viant/endly/model/transfer"
	"github.com/viant/endly/model/transformer"
	"github.com/viant/endly/model/transformer/graph"
)

type Session struct {
	baseURL           string
	bundle            transfer.Bundle
	workflowLocations map[string]string
	taskIndex         map[string]int
	subworkflow       []string
	templates         []string
	options           *transformer.Options
}

func (s *Session) workflowId() string {
	return url.Join(s.options.ProjectID, s.options.URI)
}

func (s *Session) newWorkflow(workflowNode *graph.Node) *transfer.Workflow {
	if s.options.URI == "" {
		s.options.URI = workflowNode.Name
	}
	s.bundle.Workflow = &transfer.Workflow{
		Name:      workflowNode.Name,
		URI:       s.options.URI,
		Template:  s.options.Template,
		ID:        s.workflowId(),
		ProjectID: s.bundle.Project.ID}
	return s.bundle.Workflow
}

func newSession(options *transformer.Options, URL string) *Session {
	baseURL, _ := url.Split(URL, file.Scheme)

	projectID := options.ProjectID
	if projectID == "" {
		var ancestorURL string
		ancestorURL, projectID = url.Split(baseURL, file.Scheme)
		if projectID == "e2e" {
			_, projectID = url.Split(ancestorURL, file.Scheme)
		}
		options.ProjectID = projectID
	}

	return &Session{
		bundle:            transfer.Bundle{Project: &transfer.Project{ID: projectID, Name: projectID}, Templates: map[string][]*transfer.Bundle{}},
		options:           options,
		baseURL:           baseURL,
		taskIndex:         make(map[string]int),
		workflowLocations: make(map[string]string),
	}
}
