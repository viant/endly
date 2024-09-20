package project

import (
	"context"
	"embed"
	"fmt"
	"github.com/viant/datly"
	"github.com/viant/datly/repository"
	"github.com/viant/datly/repository/contract"
	"github.com/viant/xdatly/handler/response"
	"github.com/viant/xdatly/types/core"
	"github.com/viant/xdatly/types/custom/dependency/checksum"
	"reflect"
)

func init() {
	core.RegisterType("project", "ProjectInput", reflect.TypeOf(ProjectInput{}), checksum.GeneratedTime)
	core.RegisterType("project", "ProjectOutput", reflect.TypeOf(ProjectOutput{}), checksum.GeneratedTime)
}

//go:embed project/*.sql
var ProjectFS embed.FS

type ProjectInput struct {
}

type ProjectOutput struct {
	response.Status `parameter:",kind=output,in=status"`
	Data            []*ProjectView     `parameter:",kind=output,in=view" view:"project" sql:"uri=project/project_view.sql"`
	Metrics         []*response.Metric `parameter:",kind=output,in=metrics"`
}

type ProjectView struct {
	Id          string  `sqlx:"SessionID"`
	Name        string  `sqlx:"NAME"`
	Description *string `sqlx:"DESCRIPTION"`
}

var ProjectPathURI = "/v1/api/endly/project"

func DefineProjectComponent(ctx context.Context, srv *datly.Service) error {
	aComponent, err := repository.NewComponent(
		contract.NewPath("GET", ProjectPathURI),
		repository.WithResource(srv.Resource()),
		repository.WithContract(
			reflect.TypeOf(ProjectInput{}),
			reflect.TypeOf(ProjectOutput{}), &ProjectFS))

	if err != nil {
		return fmt.Errorf("failed to create Project component: %w", err)
	}
	if err := srv.AddComponent(ctx, aComponent); err != nil {
		return fmt.Errorf("failed to add Project component: %w", err)
	}
	return nil
}
