package model

import (
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"strings"
)



const (
	//CatchTask  represent a task name that execute if error occurred and defined
	CatchTask = "catch"
	//DeferredTask represent a task name that always execute if defined
	DeferredTask = "defer"
	//ExplicitModelAttributePrefix represent model attribute prefix
	ExplicitModelAttributePrefix = ":"
)

type MapEntry struct {
	Key   string      `description:"preserved order map entry key"`
	Value interface{} `description:"preserved order map entry value"`
}

type Pipelines struct {
	baseURL  string
	Init     interface{}
	Post     interface{}
	Defaults map[string]interface{}
	Pipeline []*MapEntry
}

func (p Pipelines) updateReservedAttributes(aMap map[string]interface{}) {
	for _, key := range []string{"action", "workflow"} {
		if val, ok := aMap[key]; ok {
			if _, has := aMap[ExplicitModelAttributePrefix+key]; has {
				continue
			}
			delete(aMap, key)
			aMap[ExplicitModelAttributePrefix+key] = val
		}
	}
}

func (p Pipelines) loadRequest(attributes, actionData map[string]interface{}) error {
	if request, ok := attributes["request"]; ok && toolbox.IsString(request) {
		request := toolbox.AsString(attributes["request"])
		var requestMap = map[string]interface{}{}
		if err := util.DecodeMap(p.baseURL, request, requestMap); err != nil {
			return err
		}
		util.Append(actionData, requestMap, false)
	}
	return nil
}

func (p Pipelines) asVariables(source interface{}) ([]interface{}, error) {
	if source == nil {
		return nil, nil
	}
	var result = make([]interface{}, 0)
	variables, err := GetVariables(p.baseURL, source)
	if err != nil {
		return nil, err
	}
	err = toolbox.DefaultConverter.AssignConverted(&result, variables)
	return result, err
}

func (p Pipelines) split(source interface{}) (attributes, actionData map[string]interface{}, err error) {
	aMap, err := util.NormalizeMap(source, false)
	attributes = make(map[string]interface{})
	actionData = make(map[string]interface{})
	p.updateReservedAttributes(aMap)

	for k, v := range aMap {
		if strings.HasPrefix(k, ExplicitModelAttributePrefix) {
			attributes[strings.ToLower(string(k[1:]))] = v
			continue
		}
		attributes[k] = v
		actionData[k] = v
	}
	if err = p.loadRequest(attributes, actionData); err != nil {
		return nil, nil, err
	}

	if value, ok := attributes["init"]; ok {
		variables, err := p.asVariables(value);
		if err != nil {
			if _, has := actionData["init"]; !has {
				return nil, nil, err
			} else {
				delete(attributes, "init")
			}
		} else {
			attributes["init"] = variables
		}
	}

	if value, ok := attributes["post"]; ok {
		variables, err := p.asVariables(value);
		if err != nil {
			if _, has := actionData["post"]; !has {
				return nil, nil, err
			} else {
				delete(attributes, "post")
			}
		} else {
			attributes["post"] = variables
		}
	}
	return attributes, actionData, err
}

func (p *Pipelines) AsWorkflow(name string, baseURL string) (*Workflow, error) {
	p.baseURL = baseURL
	var result = &Workflow{
		AbstractNode: &AbstractNode{
			Name: name,
		},
		TasksNode: &TasksNode{
			Tasks: []*Task{},
		},
	}
	var err error
	if p.Init != nil {
		if result.AbstractNode.Init, err = GetVariables(p.baseURL, p.Init); err != nil {
			return nil, err
		}
	}
	if p.Post != nil {
		if result.AbstractNode.Post, err = GetVariables(p.baseURL, p.Post); err != nil {
			return nil, err
		}
	}
	root := p.buildTask("", map[string]interface{}{})
	tagID := name
	for _, entry := range p.Pipeline {
		if err = p.buildWorkflowNodes(entry.Key, entry.Value, root, tagID); err != nil {
			return nil, err
		}
	}
	p.normalize(root.TasksNode)

	if len(root.Tasks) > 0 {
		result.TasksNode = root.TasksNode
	} else {
		result.TasksNode = &TasksNode{
			Tasks: []*Task{root},
		}
	}
	return result, nil

}

func (p *Pipelines) normalize(node *TasksNode) {
	for _, task := range node.Tasks {
		if task.Name == CatchTask {
			node.OnErrorTask = task.Name
		}
		if task.Name == DeferredTask {
			node.DeferredTask = task.Name
		}
		p.normalize(task.TasksNode)
	}
}

func (p *Pipelines) buildTask(name string, source interface{}) *Task {
	var task = &Task{}
	toolbox.DefaultConverter.AssignConverted(task, source)
	task.Actions = []*Action{}
	task.AbstractNode = &AbstractNode{}
	task.TasksNode = &TasksNode{
		Tasks: []*Task{},
	}
	task.Name = name
	return task
}

func isActionNode(attributes map[string]interface{}) bool {
	if len(attributes) == 0 {
		return false
	}
	_, action := attributes["action"]
	_, workflow := attributes["workflow"]
	return action || workflow
}

func (p *Pipelines) buildAction(name string, attributes, actionData map[string]interface{}, tagId string) (*Action, error) {
	var result = &Action{
		AbstractNode:   &AbstractNode{},
		ServiceRequest: &ServiceRequest{},
		Repeater:       &Repeater{},
	}
	util.Append(actionData, p.Defaults, false)
	if action, ok := attributes["action"]; ok {
		attributes["request"], _ = util.NormalizeMap(actionData, false)
		selector := ActionSelector(toolbox.AsString(action))
		attributes["service"] = selector.Service()
		attributes["action"] = selector.Action()
	} else {
		workflow := toolbox.AsString(attributes["workflow"])
		attributes["action"] = "run"
		selector := WorkflowSelector(workflow)
		attributes["request"] = map[string]interface{}{
			"params": actionData,
			"tasks":  selector.Tasks(),
			"URL":    selector.URL(),
		}

	}
	if err := toolbox.DefaultConverter.AssignConverted(result, attributes); err != nil {
		return nil, err
	}
	result.Init()
	if result.Name == "" {
		result.Name = name
	}
	if result.Tag == "" {
		result.Tag = name
	}
	if result.TagID == "" {
		result.TagID = tagId
	}
	if result.TagID == "" {
		result.TagID = name
	}
	return result, nil
}

func (p *Pipelines) hasActionNode(source interface{}) bool {
	if source == nil {
		return false
	}
	var result = false
	attributes, _ := util.NormalizeMap(source, false)

	if isActionNode(attributes) {
		return true
	}

	toolbox.ProcessMap(attributes, func(key, value interface{}) bool {
		if !(toolbox.IsMap(value) || toolbox.IsStruct(value) || toolbox.IsSlice(value)) {
			return true
		}
		if p.hasActionNode(value) {
			result = true
			return false
		}
		return true
	})
	return result
}

func (p *Pipelines) buildWorkflowNodes(name string, source interface{}, parentTask *Task, tagID string) error {
	attributes, actionData, err := p.split(source)
	if err != nil {
		return err
	}
	if isActionNode(attributes) {
		action, err := p.buildAction(name, attributes, actionData, tagID)
		if err != nil {
			return err
		}
		task := parentTask
		if parentTask.Description == "" {
			task = p.buildTask(name, map[string]interface{}{})
			parentTask.Tasks = append(parentTask.Tasks, task)
		}
		task.Actions = append(task.Actions, action)

		return nil
	}

	if !p.hasActionNode(attributes) {
		return nil
	}

	task := p.buildTask(name, source)
	parentTask.Tasks = append(parentTask.Tasks, task)

	var buildErr error
	if err := toolbox.ProcessMap(source, func(key, value interface{}) bool {
		if !toolbox.IsSlice(value) {
			return true
		}
		buildErr = p.buildWorkflowNodes(toolbox.AsString(key), value, task, tagID+"_"+task.Name)
		if buildErr != nil {
			return false
		}
		return true
	}); err != nil {
		return err
	}
	return buildErr

}
