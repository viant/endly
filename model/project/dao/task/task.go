package task

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
	core.RegisterType("task", "TaskInput", reflect.TypeOf(TaskInput{}), checksum.GeneratedTime)
	core.RegisterType("task", "TaskOutput", reflect.TypeOf(TaskOutput{}), checksum.GeneratedTime)
}

//go:embed task/*.sql
var TaskFS embed.FS

type TaskInput struct {
	ProjectID  string        `parameter:",kind=path,in=ProjectID" predicate:"exists,group=0,t,WORKFLOW_ID,w,WORKFLOW,SessionID,PROJECT_ID"`
	ID         []string      `parameter:",kind=form,in=id" predicate:"in,group=0,a,SessionID"`
	WorkflowID string        `parameter:",kind=form,in=wid" predicate:"equal,group=0,t,WORKFLOW_ID"`
	Has        *TaskInputHas `setMarker:"true" format:"-" sqlx:"-" diff:"-"`
}

type TaskInputHas struct {
	ProjectID  bool
	ID         bool
	WorkflowID bool
}

type TaskOutput struct {
	response.Status `parameter:",kind=output,in=status"`
	Data            []*TaskView        `parameter:",kind=output,in=view" view:"task" sql:"uri=task/task_view.sql"`
	Metrics         []*response.Metric `parameter:",kind=output,in=metrics"`
}

type TaskView struct {
	Id            string  `sqlx:"SessionID"`
	WorkflowId    string  `sqlx:"WORKFLOW_ID"`
	ParentId      *string `sqlx:"PARENT_ID"`
	Position      *int    `sqlx:"POSITION"`
	Tag           string  `sqlx:"TAG"`
	Init          *string `sqlx:"INIT"`
	Post          *string `sqlx:"POST"`
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
	Data          *string `sqlx:"DATA"`
	Variables     *string `sqlx:"VARIABLES"`
	Extracts      *string `sqlx:"EXTRACTS"`
	SleepTimeMs   *int    `sqlx:"SLEEP_TIME_MS"`
	ThinkTimeMs   *int    `sqlx:"THINK_TIME_MS"`
	Logging       *bool   `sqlx:"LOGGING"`
	RepeatRun     *int    `sqlx:"REPEAT_RUN"`
	InstanceIndex *int    `sqlx:"INSTANCE_INDEX"`
	InstanceTag   *string `sqlx:"INSTANCE_TAG"`
}

var TaskPathURI = "/v1/api/endly/task/{ProjectID}"

func DefineTaskComponent(ctx context.Context, srv *datly.Service) error {
	aComponent, err := repository.NewComponent(
		contract.NewPath("GET", TaskPathURI),
		repository.WithResource(srv.Resource()),
		repository.WithContract(
			reflect.TypeOf(TaskInput{}),
			reflect.TypeOf(TaskOutput{}), &TaskFS))

	if err != nil {
		return fmt.Errorf("failed to create Task component: %w", err)
	}
	if err := srv.AddComponent(ctx, aComponent); err != nil {
		return fmt.Errorf("failed to add Task component: %w", err)
	}
	return nil
}
