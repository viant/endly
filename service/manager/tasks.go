package manager

import (
	"fmt"
	"strings"

	"github.com/viant/endly/model"
	"github.com/viant/endly/service/workflow"
	"github.com/viant/toolbox"
)

// ListTasks reflects the decoded workflow, including template expansion performed
// by the ordinary Endly loader. It performs no setup or test actions.
func (s *Service) ListTasks(request *ListTasksRequest) (*ListTasksResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("request was nil")
	}
	session, err := s.lookup(request.SessionID)
	if err != nil {
		return nil, err
	}
	session.mu.RLock()
	_, loaded := session.resolveLoaded(request.Workflow)
	if loaded == nil {
		session.mu.RUnlock()
		return nil, fmt.Errorf("workflow %q was not loaded", request.Workflow)
	}
	name := loaded.Name
	session.mu.RUnlock()
	workflow, err := session.workflow.Workflow(name)
	if err != nil {
		return nil, err
	}
	tasks := workflow.TasksNode
	if request.Path != "" {
		tasks, err = tasks.SelectWithMode(model.TasksSelector(request.Path), "path")
		if err != nil {
			return nil, err
		}
	}
	described, err := describeTasks(tasks.Tasks, "", request.ExpandWorkflows, 0)
	if err != nil {
		return nil, err
	}
	return &ListTasksResponse{SessionID: request.SessionID, Workflow: request.Workflow, Tasks: described}, nil
}

func describeTasks(tasks model.Tasks, prefix string, expand bool, depth int) ([]*TaskInfo, error) {
	if depth > 16 {
		return nil, fmt.Errorf("nested workflow discovery exceeded 16 levels")
	}
	result := make([]*TaskInfo, 0, len(tasks))
	for _, task := range tasks {
		path := task.Name
		if prefix != "" {
			path = prefix + "." + path
		}
		info := &TaskInfo{Name: task.Name, Path: path}
		if task.TasksNode != nil {
			var err error
			info.Tasks, err = describeTasks(task.Tasks, path, expand, depth)
			if err != nil {
				return nil, err
			}
		}
		instances := map[string]*TemplateInstanceInfo{}
		for _, action := range task.Actions {
			item := TaskActionInfo{Name: action.Name, Service: action.Service, Action: action.Action, TagID: action.TagID, Skip: action.Skip}
			info.Actions = append(info.Actions, item)
			if expand && action.Service == "workflow" && action.Action == "run" {
				child := &workflow.RunRequest{}
				if err := toolbox.DefaultConverter.AssignConverted(child, action.Request); err != nil {
					return nil, err
				}
				if child.Inlined != nil {
					if err := child.Init(); err != nil {
						return nil, err
					}
					source := child.AssetURL
					base, _ := toolbox.URLSplit(source)
					model, err := child.AsWorkflow(child.Name, base)
					if err != nil {
						return nil, err
					}
					nested, err := describeTasks(model.Tasks, "", expand, depth+1)
					if err != nil {
						return nil, err
					}
					info.Workflows = append(info.Workflows, &WorkflowTaskInfo{Name: model.Name, Source: source, Tasks: nested})
				}
			}
			if strings.TrimSpace(action.TagID) == "" || action.TagIndex == "" {
				continue
			}
			instance := instances[action.TagID]
			if instance == nil {
				instance = &TemplateInstanceInfo{TagID: action.TagID, Index: action.TagIndex, Tag: action.Tag, Description: action.TagDescription}
				instances[action.TagID] = instance
				info.Instances = append(info.Instances, instance)
			}
			instance.Actions = append(instance.Actions, item)
		}
		result = append(result, info)
	}
	return result, nil
}
