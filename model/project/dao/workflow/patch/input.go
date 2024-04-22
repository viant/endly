package patch

import (
	"embed"
	"github.com/viant/xdatly/types/core"
	"github.com/viant/xdatly/types/custom/checksum"
	"reflect"
)

//go:embed workflow/*.sql
var WorkflowPatchFS embed.FS

func init() {
	core.RegisterType(PackageName, "Input", reflect.TypeOf(Input{}), checksum.GeneratedTime)

}

type Input struct {
	Workflow *Workflow `parameter:",kind=body"`

	/*
	   ? SELECT ARRAY_AGG(Id) AS Values FROM  `/` LIMIT 1
	*/
	CurWorkflowId *struct{ Values []string } `parameter:",kind=param,in=Workflow,dataType=workflow/patch.Workflow" codec:"structql,uri=workflow/cur_workflow_id.sql"`

	/*
	   ? SELECT ARRAY_AGG(Id) AS Values FROM  `/Task` LIMIT 1
	*/
	CurWorkflowTaskId *struct{ Values []string } `parameter:",kind=param,in=Workflow,dataType=workflow/patch.Workflow" codec:"structql,uri=workflow/cur_workflow_task_id.sql"`

	/*
	    ? SELECT *
	     FROM TASK
	   WHERE $criteria.In("SessionID", $CurWorkflowTaskId.Values)
	*/
	CurTask []*Task `parameter:",kind=view,in=Task" view:"Task" sql:"uri=workflow/cur_task.sql"`

	/*
	   ? SELECT ARRAY_AGG(Id) AS Values FROM  `/Asset` LIMIT 1
	*/
	CurWorkflowAssetId *struct{ Values []string } `parameter:",kind=param,in=Workflow,dataType=workflow/patch.Workflow" codec:"structql,uri=workflow/cur_workflow_asset_id.sql"`

	/*
	    ? SELECT *
	     FROM ASSET
	   WHERE $criteria.In("SessionID", $CurWorkflowAssetId.Values)
	*/
	CurAsset []*Asset `parameter:",kind=view,in=Asset" view:"Asset" sql:"uri=workflow/cur_asset.sql"`

	/*
	   ? SELECT ARRAY_AGG(Id) AS Values FROM  `/Project` LIMIT 1
	*/
	CurWorkflowProjectId *struct{ Values []string } `parameter:",kind=param,in=Workflow,dataType=workflow/patch.Workflow" codec:"structql,uri=workflow/cur_workflow_project_id.sql"`

	/*
	    ? SELECT *
	     FROM PROJECT
	   WHERE $criteria.In("SessionID", $CurWorkflowProjectId.Values)
	*/
	CurProject []*Project `parameter:",kind=view,in=Project" view:"Project" sql:"uri=workflow/cur_project.sql"`

	/*
	    ? select * from WORKFLOW
	   WHERE $criteria.In("SessionID", $CurWorkflowId.Values)
	*/
	CurWorkflow *Workflow `parameter:",kind=view,in=Workflow" view:"Workflow" sql:"uri=workflow/cur_workflow.sql"`
}
