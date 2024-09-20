package patch

import (
	"github.com/viant/xdatly/types/core"
	"github.com/viant/xdatly/types/custom/dependency/checksum"
	"reflect"
)

var PackageName = "workflow/patch"

var Types map[string]reflect.Type

func init() {
	core.RegisterType(PackageName, "Task", reflect.TypeOf(Task{}), checksum.GeneratedTime)
	core.RegisterType(PackageName, "Asset", reflect.TypeOf(Asset{}), checksum.GeneratedTime)
	core.RegisterType(PackageName, "Project", reflect.TypeOf(Project{}), checksum.GeneratedTime)
	core.RegisterType(PackageName, "Workflow", reflect.TypeOf(Workflow{}), checksum.GeneratedTime)
}

type Workflow struct {
	Id            string       `sqlx:"SessionID,primaryKey" validate:"required,le(255)"`
	ParentId      *string      `sqlx:"PARENT_ID" json:",omitempty" validate:"omitempty,le(255)"`
	Position      *int         `sqlx:"POSITION" json:",omitempty"`
	Revision      *string      `sqlx:"REVISION" json:",omitempty" validate:"omitempty,le(255)"`
	Uri           string       `sqlx:"URI" validate:"required,le(65535)"`
	ProjectId     string       `sqlx:"PROJECT_ID,refTable=PROJECT,refColumn=SessionID" validate:"required,le(255)"`
	Name          string       `sqlx:"NAME" validate:"required,le(255)"`
	Init          *string      `sqlx:"INIT" json:",omitempty" validate:"omitempty,le(65535)"`
	Post          *string      `sqlx:"POST" json:",omitempty" validate:"omitempty,le(65535)"`
	Template      *string      `sqlx:"TEMPLATE" json:",omitempty" validate:"omitempty,le(255)"`
	InstanceIndex *int         `sqlx:"INSTANCE_INDEX" json:",omitempty"`
	InstanceTag   *string      `sqlx:"INSTANCE_TAG" json:",omitempty" validate:"omitempty,le(255)"`
	Task          []*Task      `sqlx:"-" on:"SessionID:Id=WORKFLOW_ID:WorkflowId" view:"Task,table=TASK" sql:"   SELECT *   FROM TASK "`
	Asset         []*Asset     `sqlx:"-" on:"SessionID:Id=WORKFLOW_ID:WorkflowId" view:"Asset,table=ASSET" sql:"   SELECT *   FROM ASSET "`
	Project       []*Project   `sqlx:"-" on:"PROJECT_ID:ProjectId=SessionID:Id" view:"Project,table=PROJECT" sql:"   SELECT *   FROM PROJECT "`
	Has           *WorkflowHas `setMarker:"true" format:"-" sqlx:"-" diff:"-"`
}

type Task struct {
	Id            string   `sqlx:"SessionID,primaryKey" validate:"required,le(255)"`
	WorkflowId    string   `sqlx:"WORKFLOW_ID" validate:"required,le(255)"`
	ParentId      *string  `sqlx:"PARENT_ID" json:",omitempty" validate:"omitempty,le(255)"`
	Position      *int     `sqlx:"POSITION" json:",omitempty"`
	Tag           string   `sqlx:"TAG" validate:"required,le(255)"`
	Init          *string  `sqlx:"INIT" json:",omitempty" validate:"omitempty,le(65535)"`
	Post          *string  `sqlx:"POST" json:",omitempty" validate:"omitempty,le(65535)"`
	Description   *string  `sqlx:"DESCRIPTION" json:",omitempty" validate:"omitempty,le(65535)"`
	WhenExpr      *string  `sqlx:"WHEN_EXPR" json:",omitempty" validate:"omitempty,le(65535)"`
	ExitExpr      *string  `sqlx:"EXIT_EXPR" json:",omitempty" validate:"omitempty,le(65535)"`
	OnError       *string  `sqlx:"ON_ERROR" json:",omitempty" validate:"omitempty,le(65535)"`
	Deferred      *string  `sqlx:"DEFERRED" json:",omitempty" validate:"omitempty,le(65535)"`
	Service       *string  `sqlx:"SERVICE" json:",omitempty" validate:"omitempty,le(255)"`
	Action        *string  `sqlx:"ACTION" json:",omitempty" validate:"omitempty,le(255)"`
	Input         *string  `sqlx:"INPUT" json:",omitempty" validate:"omitempty,le(65535)"`
	InputUri      *string  `sqlx:"INPUT_URI" json:",omitempty" validate:"omitempty,le(65535)"`
	Async         *int     `sqlx:"ASYNC" json:",omitempty"`
	SkipExpr      *string  `sqlx:"SKIP_EXPR" json:",omitempty" validate:"omitempty,le(65535)"`
	Fail          *int     `sqlx:"FAIL" json:",omitempty"`
	IsTemplate    *int     `sqlx:"IS_TEMPLATE" json:",omitempty"`
	SubPath       *string  `sqlx:"SUB_PATH" json:",omitempty" validate:"omitempty,le(255)"`
	RangeExpr     *string  `sqlx:"RANGE_EXPR" json:",omitempty" validate:"omitempty,le(65535)"`
	Data          *string  `sqlx:"DATA" json:",omitempty" validate:"omitempty,le(65535)"`
	Variables     *string  `sqlx:"VARIABLES" json:",omitempty" validate:"omitempty,le(65535)"`
	Extracts      *string  `sqlx:"EXTRACTS" json:",omitempty" validate:"omitempty,le(65535)"`
	SleepTimeMs   *int     `sqlx:"SLEEP_TIME_MS" json:",omitempty"`
	ThinkTimeMs   *int     `sqlx:"THINK_TIME_MS" json:",omitempty"`
	Logging       *int     `sqlx:"LOGGING" json:",omitempty"`
	RepeatRun     *int     `sqlx:"REPEAT_RUN" json:",omitempty"`
	InstanceIndex *int     `sqlx:"INSTANCE_INDEX" json:",omitempty"`
	InstanceTag   *string  `sqlx:"INSTANCE_TAG" json:",omitempty" validate:"omitempty,le(255)"`
	Has           *TaskHas `setMarker:"true" format:"-" sqlx:"-" diff:"-"`
}

type TaskHas struct {
	Id            bool
	WorkflowId    bool
	ParentId      bool
	Position      bool
	Tag           bool
	Init          bool
	Post          bool
	Description   bool
	WhenExpr      bool
	ExitExpr      bool
	OnError       bool
	Deferred      bool
	Service       bool
	Action        bool
	Input         bool
	InputUri      bool
	Async         bool
	SkipExpr      bool
	Fail          bool
	IsTemplate    bool
	SubPath       bool
	RangeExpr     bool
	Data          bool
	Variables     bool
	Extracts      bool
	SleepTimeMs   bool
	ThinkTimeMs   bool
	Logging       bool
	RepeatRun     bool
	InstanceIndex bool
	InstanceTag   bool
}

type Asset struct {
	Id            string    `sqlx:"SessionID,primaryKey" validate:"required,le(255)"`
	Location      string    `sqlx:"LOCATION" validate:"required,le(255)"`
	Description   *string   `sqlx:"DESCRIPTION" json:",omitempty" validate:"omitempty,le(65535)"`
	WorkflowId    string    `sqlx:"WORKFLOW_ID,refTable=WORKFLOW,refColumn=SessionID" validate:"required,le(255)"`
	IsDir         *int      `sqlx:"IS_DIR" json:",omitempty"`
	Template      *string   `sqlx:"TEMPLATE" json:",omitempty" validate:"omitempty,le(255)"`
	InstanceIndex *int      `sqlx:"INSTANCE_INDEX" json:",omitempty"`
	InstanceTag   *string   `sqlx:"INSTANCE_TAG" json:",omitempty" validate:"omitempty,le(255)"`
	Position      *int      `sqlx:"POSITION" json:",omitempty"`
	Source        *string   `sqlx:"SOURCE" json:",omitempty" validate:"omitempty,le(65535)"`
	Format        *string   `sqlx:"FORMAT" json:",omitempty" validate:"omitempty,le(255)"`
	Codec         *string   `sqlx:"CODEC" json:",omitempty" validate:"omitempty,le(255)"`
	Has           *AssetHas `setMarker:"true" format:"-" sqlx:"-" diff:"-"`
}

type AssetHas struct {
	Id            bool
	Location      bool
	Description   bool
	WorkflowId    bool
	IsDir         bool
	Template      bool
	InstanceIndex bool
	InstanceTag   bool
	Position      bool
	Source        bool
	Format        bool
	Codec         bool
}

type Project struct {
	Id          string      `sqlx:"SessionID,primaryKey" validate:"required,le(255)"`
	Name        string      `sqlx:"NAME" validate:"required,le(255)"`
	Description *string     `sqlx:"DESCRIPTION" json:",omitempty" validate:"omitempty,le(65535)"`
	Has         *ProjectHas `setMarker:"true" format:"-" sqlx:"-" diff:"-"`
}

type ProjectHas struct {
	Id          bool
	Name        bool
	Description bool
}

type WorkflowHas struct {
	Id            bool
	ParentId      bool
	Position      bool
	Revision      bool
	Uri           bool
	ProjectId     bool
	Name          bool
	Init          bool
	Post          bool
	Template      bool
	InstanceIndex bool
	InstanceTag   bool
	Task          bool
	Asset         bool
	Project       bool
}
