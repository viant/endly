package patch

import (
	"context"
	"github.com/viant/xdatly/handler"
	"github.com/viant/xdatly/types/core"
	"github.com/viant/xdatly/types/custom/checksum"
	"reflect"
)

func init() {
	core.RegisterType(PackageName, "Handler", reflect.TypeOf(Handler{}), checksum.GeneratedTime)
}

type Handler struct{}

func (h *Handler) Exec(ctx context.Context, sess handler.Session) (interface{}, error) {
	input := &Input{}
	if err := sess.Stater().Into(ctx, input); err != nil {
		return nil, err
	}
	sql, err := sess.Db()
	if err != nil {
		return nil, err
	}
	workflow := input.Workflow
	curTask := input.CurTask
	curAsset := input.CurAsset
	curProject := input.CurProject
	curWorkflow := input.CurWorkflow

	curWorkflowById := WorkflowSlice([]*Workflow{curWorkflow}).IndexById()
	curTaskById := TaskSlice(curTask).IndexById()
	curAssetById := AssetSlice(curAsset).IndexById()
	curProjectById := ProjectSlice(curProject).IndexById()

	if workflow != nil {

		if curWorkflowById.Has(workflow.Id) == true {
			if err = sql.Update("WORKFLOW", workflow); err != nil {
				return nil, err
			}
		} else {
			if err = sql.Insert("WORKFLOW", workflow); err != nil {
				return nil, err
			}
		}

		for _, recTask := range workflow.Task {

			recTask.WorkflowId = workflow.Id
			if curTaskById.Has(recTask.Id) == true {
				if err = sql.Update("TASK", recTask); err != nil {
					return nil, err
				}
			} else {
				if err = sql.Insert("TASK", recTask); err != nil {
					return nil, err
				}
			}
		}

		for _, recAsset := range workflow.Asset {

			recAsset.WorkflowId = workflow.Id
			if curAssetById.Has(recAsset.Id) == true {
				if err = sql.Update("ASSET", recAsset); err != nil {
					return nil, err
				}
			} else {
				if err = sql.Insert("ASSET", recAsset); err != nil {
					return nil, err
				}
			}
		}

		for _, recProject := range workflow.Project {

			recProject.Id = workflow.ProjectId
			if curProjectById.Has(recProject.Id) == true {
				if err = sql.Update("PROJECT", recProject); err != nil {
					return nil, err
				}
			} else {
				if err = sql.Insert("PROJECT", recProject); err != nil {
					return nil, err
				}
			}
		}
	}

	response := Output{}
	response.Data = input.Workflow
	return response, nil

}
