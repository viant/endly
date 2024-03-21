package graph

import (
	"context"
	"github.com/viant/afs"
	"github.com/viant/afs/file"
	"github.com/viant/afs/storage"
	"github.com/viant/afs/url"
	"gopkg.in/yaml.v3"
	"strings"
)

type Service struct {
	fs afs.Service
}

func (s *Service) LoadWorkflow(ctx context.Context, URL string, opts ...storage.Option) (*Node, error) {
	data, err := s.fs.DownloadWithURL(ctx, URL, opts...)
	if err != nil {
		return nil, err
	}
	node := yaml.Node{}
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}
	_, name := url.Split(URL, file.Scheme)
	if index := strings.LastIndex(name, "."); index != -1 {
		name = name[:index]
	}
	workflowNode := NewWorkflowNode(name, node.Content[0])
	return workflowNode, nil
}

func New() *Service {
	return &Service{fs: afs.New()}
}
