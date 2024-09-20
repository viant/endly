package workflow

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
	core.RegisterType("workflow", "WorkflowInput", reflect.TypeOf(WorkflowInput{}), checksum.GeneratedTime)
	core.RegisterType("workflow", "WorkflowOutput", reflect.TypeOf(WorkflowOutput{}), checksum.GeneratedTime)
}

//go:embed workflow/*.sql
var WorkflowFS embed.FS

type WorkflowInput struct {
	ProjectID      string            `parameter:",kind=path,in=ProjectID"`
	ID             []string          `parameter:",kind=form,in=id" predicate:"in,group=0,w,SessionID"`
	Name           []string          `parameter:",kind=form,in=name" predicate:"in,group=0,w,NAME"`
	Template       []string          `parameter:",kind=form,in=template" predicate:"in,group=0,w,TEMPLATE"`
	TemplateIsNull bool              `parameter:",kind=form,in=standalone" predicate:"is_null,group=0,w,TEMPLATE"`
	InstanceTag    []string          `parameter:",kind=form,in=template_tag" predicate:"in,group=0,w,INSTANCE_TAG"`
	InstanceIndex  []int             `parameter:",kind=form,in=template_index" predicate:"in,group=0,w,INSTANCE_INDEX"`
	Has            *WorkflowInputHas `setMarker:"true" format:"-" sqlx:"-" diff:"-"`
}

type WorkflowInputHas struct {
	ProjectID      bool
	ID             bool
	Name           bool
	Template       bool
	TemplateIsNull bool
	InstanceTag    bool
	InstanceIndex  bool
}

type WorkflowOutput struct {
	Data    []*WorkflowView    `parameter:",kind=output,in=view" view:"workflow" sql:"uri=workflow/workflow_view.sql"`
	Metrics []*response.Metric `parameter:",kind=output,in=metrics"`
}

type WorkflowView struct {
	Id            string      `sqlx:"SessionID"`
	ParentId      *string     `sqlx:"PARENT_ID"`
	Position      *int        `sqlx:"POSITION"`
	Revision      *string     `sqlx:"REVISION"`
	Uri           string      `sqlx:"URI"`
	ProjectId     string      `sqlx:"PROJECT_ID"`
	Name          string      `sqlx:"NAME"`
	Init          *string     `jsonx:"inline" sqlx:"INIT"`
	Post          *string     `jsonx:"inline" sqlx:"POST"`
	Template      *string     `sqlx:"TEMPLATE"`
	InstanceIndex *int        `sqlx:"INSTANCE_INDEX"`
	InstanceTag   *string     `sqlx:"INSTANCE_TAG"`
	Description   *string     `sqlx:"DESCRIPTION"`
	Task          []*TaskView `on:"Id:SessionID=WorkflowId:WORKFLOW_ID" sql:"uri=workflow/task.sql"`
	Asset         *AssetView  `on:"Id:SessionID=WorkflowId:WORKFLOW_ID" sql:"uri=workflow/asset.sql"`
}

type TaskView struct {
	Id            string  `sqlx:"SessionID"`
	WorkflowId    string  `sqlx:"WORKFLOW_ID"`
	ParentId      *string `sqlx:"PARENT_ID"`
	Position      *int    `sqlx:"POSITION"`
	Tag           string  `sqlx:"TAG"`
	Init          *string `jsonx:"inline" sqlx:"INIT"`
	Post          *string `jsonx:"inline" sqlx:"POST"`
	Description   *string `sqlx:"DESCRIPTION"`
	WhenExpr      *string `sqlx:"WHEN_EXPR"`
	ExitExpr      *string `sqlx:"EXIT_EXPR"`
	OnError       *string `sqlx:"ON_ERROR"`
	Deferred      *string `sqlx:"DEFERRED"`
	Service       *string `sqlx:"SERVICE"`
	Action        *string `sqlx:"ACTION"`
	Input         *string `sqlx:"INPUT"`
	InputUri      *string `sqlx:"INPUT_URI"`
	Async         *bool   `sqlx:"ASYNC"`
	SkipExpr      *string `sqlx:"SKIP_EXPR"`
	Fail          *bool   `sqlx:"FAIL"`
	IsTemplate    *bool   `sqlx:"IS_TEMPLATE"`
	SubPath       *string `sqlx:"SUB_PATH"`
	RangeExpr     *string `sqlx:"RANGE_EXPR"`
	Data          *string `jsonx:"inline" sqlx:"DATA"`
	Variables     *string `jsonx:"inline" sqlx:"VARIABLES"`
	Extracts      *string `jsonx:"inline" sqlx:"EXTRACTS"`
	SleepTimeMs   *int    `sqlx:"SLEEP_TIME_MS"`
	ThinkTimeMs   *int    `sqlx:"THINK_TIME_MS"`
	Logging       *bool   `sqlx:"LOGGING"`
	RepeatRun     *int    `sqlx:"REPEAT_RUN"`
	InstanceIndex *int    `sqlx:"INSTANCE_INDEX"`
	InstanceTag   *string `sqlx:"INSTANCE_TAG"`
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
	Format        *string `sqlx:"FORMAT"`
	Codec         *string `sqlx:"CODEC"`
}

var WorkflowPathURI = "/v1/api/endly/workflow/{ProjectID}"

func DefineWorkflowComponent(ctx context.Context, srv *datly.Service) error {
	aComponent, err := repository.NewComponent(
		contract.NewPath("GET", WorkflowPathURI),
		repository.WithResource(srv.Resource()),
		repository.WithContract(
			reflect.TypeOf(WorkflowInput{}),
			reflect.TypeOf(WorkflowOutput{}), &WorkflowFS))

	if err != nil {
		return fmt.Errorf("failed to create Workflow component: %w", err)
	}
	if err := srv.AddComponent(ctx, aComponent); err != nil {
		return fmt.Errorf("failed to add Workflow component: %w", err)
	}
	return nil
}
