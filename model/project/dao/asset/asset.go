package asset

import (
	"context"
	"embed"
	"fmt"
	"github.com/viant/datly"
	"github.com/viant/datly/repository"
	"github.com/viant/datly/repository/contract"
	"github.com/viant/xdatly/handler/response"
	"github.com/viant/xdatly/types/core"
	"github.com/viant/xdatly/types/custom/checksum"
	"reflect"
)

func init() {
	core.RegisterType("asset", "AssetInput", reflect.TypeOf(AssetInput{}), checksum.GeneratedTime)
	core.RegisterType("asset", "AssetOutput", reflect.TypeOf(AssetOutput{}), checksum.GeneratedTime)
}

//go:embed asset/*.sql
var AssetFS embed.FS

type AssetInput struct {
	ProjectID  string         `parameter:",kind=path,in=ProjectID" predicate:"exists,group=0,a,WORKFLOW_ID,w,WORKFLOW,SessionID,PROJECT_ID"`
	ID         []string       `parameter:",kind=form,in=id" predicate:"in,group=0,a,SessionID"`
	WorkflowID string         `parameter:",kind=form,in=wid" predicate:"equal,group=0,a,WORKFLOW_ID"`
	Has        *AssetInputHas `setMarker:"true" format:"-" sqlx:"-" diff:"-"`
}

type AssetInputHas struct {
	ProjectID  bool
	ID         bool
	WorkflowID bool
}

type AssetOutput struct {
	response.Status `parameter:",kind=output,in=status"`
	Data            []*AssetView       `parameter:",kind=output,in=view" view:"asset" sql:"uri=asset/asset_view.sql"`
	Metrics         []*response.Metric `parameter:",kind=output,in=metrics"`
}

type AssetView struct {
	Id            string  `sqlx:"SessionID"`
	Location      string  `sqlx:"LOCATION"`
	Description   *string `sqlx:"DESCRIPTION"`
	WorkflowId    string  `sqlx:"WORKFLOW_ID"`
	IsDir         *int    `sqlx:"IS_DIR"`
	Template      *string `sqlx:"TEMPLATE"`
	InstanceIndex *int    `sqlx:"INSTANCE_INDEX"`
	InstanceTag   *string `sqlx:"INSTANCE_TAG"`
	Position      *int    `sqlx:"POSITION"`
	Source        *string `sqlx:"SOURCE"`
	Format        *string `sqlx:"FORMAT"`
	Codec         *string `sqlx:"CODEC"`
}

var AssetPathURI = "/v1/api/endly/asset/{ProjectID}"

func DefineAssetComponent(ctx context.Context, srv *datly.Service) error {
	aComponent, err := repository.NewComponent(
		contract.NewPath("GET", AssetPathURI),
		repository.WithResource(srv.Resource()),
		repository.WithContract(
			reflect.TypeOf(AssetInput{}),
			reflect.TypeOf(AssetOutput{}), &AssetFS))

	if err != nil {
		return fmt.Errorf("failed to create Asset component: %w", err)
	}
	if err := srv.AddComponent(ctx, aComponent); err != nil {
		return fmt.Errorf("failed to add Asset component: %w", err)
	}
	return nil
}
