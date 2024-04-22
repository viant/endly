package loader

import (
	"github.com/viant/afs/file"
	"github.com/viant/afs/url"
	"github.com/viant/endly/model/graph"
	project "github.com/viant/endly/model/project"
	option "github.com/viant/endly/model/project/option"
)

type Session struct {
	baseURL           string
	bundle            project.Bundle
	workflowLocations map[string]string
	taskIndex         map[string]int
	subWorkflows      []string
	templates         []string
	options           *option.Options
	assets            *graph.AssetManager
}

func (s *Session) workflowId() string {
	return url.Join(s.options.ProjectID, s.options.URI)
}

func (s *Session) newWorkflow(workflowNode *graph.Node, asset *graph.Asset) *project.Workflow {
	s.options.URI = asset.URI
	s.bundle.Workflow = &project.Workflow{
		Name:      workflowNode.Name,
		URI:       s.options.URI,
		Template:  s.options.Template,
		ID:        s.workflowId(),
		ProjectID: s.bundle.Project.ID}
	return s.bundle.Workflow
}

func newSession(options *option.Options, URL string) *Session {

	if options.BaseURL == "" {
		options.BaseURL, _ = url.Split(URL, file.Scheme)
	}

	projectID := options.ProjectID
	if projectID == "" {
		var ancestorURL string
		ancestorURL, projectID = url.Split(options.BaseURL, file.Scheme)
		if projectID == "e2e" {
			_, projectID = url.Split(ancestorURL, file.Scheme)
		}
		options.ProjectID = projectID
	}
	if options.Assets == nil {
		options.Assets = graph.NewAssetManager(options.BaseURL, options.StorageOptions()...)

	}
	return &Session{
		bundle:            project.Bundle{Project: &project.Project{ID: projectID, Name: projectID}, Templates: map[string][]*project.Bundle{}},
		options:           options,
		baseURL:           options.BaseURL,
		taskIndex:         make(map[string]int),
		workflowLocations: make(map[string]string),
		assets:            options.Assets,
	}
}
