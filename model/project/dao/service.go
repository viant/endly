package dao

import (
	"context"
	"github.com/viant/datly"
	"github.com/viant/datly/view"
	"github.com/viant/endly/model/project/dao/asset"
	"github.com/viant/endly/model/project/dao/project"
	"github.com/viant/endly/model/project/dao/task"
	"github.com/viant/endly/model/project/dao/workflow"
	"github.com/viant/endly/model/project/dao/workflow/patch"
)

type Service struct {
	*datly.Service
}

func (s *Service) Init(ctx context.Context) error {
	if err := s.AddConnectors(ctx,
		view.NewConnector("endly", "mysql", "root"+":"+"dev"+"@tcp(localhost:3306)/endly?parseTime=true"),
	); err != nil {
		return err
	}
	if _, err := patch.DefineWorkflowPatchComponent(ctx, s.Service); err != nil {
		return err
	}
	if err := project.DefineProjectComponent(ctx, s.Service); err != nil {
		return err
	}
	if err := workflow.DefineWorkflowComponent(ctx, s.Service); err != nil {
		return err
	}

	if err := task.DefineTaskComponent(ctx, s.Service); err != nil {
		return err
	}
	if err := asset.DefineAssetComponent(ctx, s.Service); err != nil {
		return err
	}
	return nil

}

func New(ctx context.Context) (*Service, error) {
	datlyService, err := datly.New(ctx)
	if err != nil {
		return nil, err
	}
	return &Service{Service: datlyService}, nil
}
