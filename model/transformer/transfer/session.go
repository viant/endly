package transfer

import (
	"github.com/viant/afs/file"
	"github.com/viant/afs/url"
	"github.com/viant/endly/model/graph"
	"github.com/viant/endly/model/transfer"
	"github.com/viant/endly/model/transformer"
)

type Session struct {
	baseURL           string
	bundle            transfer.Bundle
	workflowLocations map[string]string
	taskIndex         map[string]int
	subworkflow       []string
	templates         []string
	options           *transformer.Options
	assets             *graph.AssetManager
}

func (s *Session) workflowId() string {
	return url.Join(s.options.ProjectID, s.options.URI)
}

func (s *Session) newWorkflow(workflowNode *graph.Node, asset *graph.Asset) *transfer.Workflow {
	s.options.URI = asset.URI
	s.bundle.Workflow = &transfer.Workflow{
		Name:      workflowNode.Name,
		URI:       s.options.URI,
		Template:  s.options.Template,
		ID:        s.workflowId(),
		ProjectID: s.bundle.Project.ID}
	return s.bundle.Workflow
}

func newSession(options *transformer.Options, URL string) *Session {

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
		bundle:            transfer.Bundle{Project: &transfer.Project{ID: projectID, Name: projectID}, Templates: map[string][]*transfer.Bundle{}},
		options:           options,
		baseURL:           options.BaseURL,
		taskIndex:         make(map[string]int),
		workflowLocations: make(map[string]string),
		assets: 		  options.Assets,
	}
}
