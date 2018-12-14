package model

import (
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"strings"
)

const (
	//CatchTask  represent a task name that execute if error occurred and defined
	CatchTask = "catch"

	//DeferredTask represent a task name that always execute if defined
	DeferredTask = "defer"
	//ExplicitActionAttributePrefix represent model attribute prefix
	ExplicitActionAttributePrefix  = ":"
	ExplicitRequestAttributePrefix = "@"
)

var multiActionKeys = []string{"multiaction", "async"}

type MapEntry struct {
	Key   string      `description:"preserved order map entry key"`
	Value interface{} `description:"preserved order map entry value"`
}

type InlineWorkflow struct {
	baseURL    string
	tagPathURL string
	name       string
	Init       interface{}
	Post       interface{}
	Logging    *bool
	Defaults   map[string]interface{}
	Data       map[string]interface{}
	Pipeline   []*MapEntry
	workflow   *Workflow //inline workflow from pipeline
}

func (p InlineWorkflow) updateReservedAttributes(aMap map[string]interface{}) {
	for _, key := range []string{"action", "workflow"} {
		if val, ok := aMap[key]; ok {
			if _, has := aMap[ExplicitActionAttributePrefix+key]; has {
				continue
			}
			delete(aMap, key)
			aMap[ExplicitActionAttributePrefix+key] = val
		}
	}
}

var normalizationBlacklist = map[string]bool{
	"workflow:run":     true,
	"seleniun:run":     true,
	"validator:assert": true,
}

func isNormalizableRequest(actionAttributes map[string]interface{}) bool {
	if len(actionAttributes) == 0 {
		return true
	}
	if _, ok := actionAttributes["workflow"]; ok {
		return false
	}

	action := ""
	if val, ok := actionAttributes["action"]; ok {
		action = toolbox.AsString(val)
		action = strings.Replace(action, ".", ":", 1)
	}
	if strings.Count(action, ":") == 0 {
		service := "workflow"
		if val, ok := actionAttributes["service"]; ok {
			service = toolbox.AsString(val)
		}
		action = service + ":" + action
	}
	_, has := normalizationBlacklist[action]
	return !has
}

func (p InlineWorkflow) loadRequest(actionAttributes, actionRequest map[string]interface{}, state data.Map) error {
	if request, ok := actionAttributes["request"]; ok && toolbox.IsString(request) {
		request := toolbox.AsString(actionAttributes["request"])
		requestMap, err := util.LoadMap([]string{p.tagPathURL, p.baseURL}, request)
		if err != nil {
			return err
		}

		if isNormalizableRequest(actionAttributes) {
			requestMap, err = util.NormalizeMap(requestMap, true)
			if err != nil {
				return err
			}
		}
		expanded := state.Expand(requestMap)
		dataRequest := data.Map(actionRequest)
		expanded = dataRequest.Expand(expanded)
		requestMap = toolbox.AsMap(expanded)

		if val, ok := requestMap["defaults"]; ok {
			if defaults, err := util.NormalizeMap(val, false); err == nil {
				requestMap["defaults"] = defaults
			}
		}
		util.Append(actionRequest, requestMap, false)
	}
	return nil
}

func (p InlineWorkflow) asVariables(source interface{}) ([]map[string]interface{}, error) {
	if source == nil {
		return nil, nil
	}
	var result = make([]map[string]interface{}, 0)
	variables, err := GetVariables([]string{p.tagPathURL, p.baseURL}, source)
	if err != nil {
		return nil, err
	}
	err = toolbox.DefaultConverter.AssignConverted(&result, variables)
	return result, err
}

//split splits key value pair into workflow action attribute and action request data,
// while ':' key prefix assign pair to workflow action, '@' assign to request data, if none is matched pair is assign to both
func (p InlineWorkflow) split(source interface{}, state data.Map) (map[string]interface{}, map[string]interface{}, error) {
	aMap, err := util.NormalizeMap(source, false)
	var actionAttributes = make(map[string]interface{})
	var actionRequest = make(map[string]interface{})
	p.updateReservedAttributes(aMap)

	for k, v := range aMap {
		if strings.HasPrefix(k, ExplicitActionAttributePrefix) {
			actionAttributes[strings.ToLower(string(k[1:]))] = v
			continue
		}
		if strings.HasPrefix(k, ExplicitRequestAttributePrefix) {
			actionRequest[strings.ToLower(string(k[1:]))] = v
			continue
		}
		actionAttributes[k] = v
		actionRequest[k] = v
	}
	if err = p.loadRequest(actionAttributes, actionRequest, state); err != nil {
		return nil, nil, err
	}

	if value, ok := actionAttributes["logging"]; ok {
		actionAttributes["logging"] = toolbox.AsBoolean(value)
	}

	if value, ok := actionAttributes["init"]; ok {
		variables, err := p.asVariables(value)
		if err != nil {
			if _, has := actionRequest["init"]; !has {
				return nil, nil, err
			} else {
				delete(actionAttributes, "init")
			}
		} else {
			actionAttributes["init"] = state.Expand(variables)
		}
	}

	if value, ok := actionAttributes["post"]; ok {
		variables, err := p.asVariables(value)
		if err != nil {
			if _, has := actionRequest["post"]; !has {
				return nil, nil, err
			} else {
				delete(actionAttributes, "post")
			}
		} else {
			actionAttributes["post"] = state.Expand(variables)
		}
	}
	return actionAttributes, actionRequest, err
}

func (p *InlineWorkflow) AsWorkflow(name string, baseURL string) (*Workflow, error) {
	if p.workflow != nil {
		return p.workflow, nil
	}
	p.baseURL = baseURL
	p.name = name
	if len(p.Data) == 0 {
		p.Data = make(map[string]interface{})
	}
	var workflow = &Workflow{
		AbstractNode: &AbstractNode{
			Name:    name,
			Logging: p.Logging,
		},
		TasksNode: &TasksNode{
			Tasks: []*Task{},
		},

		Data:   p.Data,
		Source: url.NewResource(toolbox.URLPathJoin(baseURL, name+".yaml")),
	}
	var err error
	if p.Init != nil {
		if workflow.AbstractNode.Init, err = GetVariables([]string{p.baseURL}, p.Init); err != nil {
			return nil, err
		}
	}
	if p.Post != nil {
		if workflow.AbstractNode.Post, err = GetVariables([]string{p.baseURL}, p.Post); err != nil {
			return nil, err
		}
	}
	root := p.buildTask("", map[string]interface{}{})
	tagID := name
	for _, entry := range p.Pipeline {
		if err = p.buildWorkflowNodes(entry.Key, entry.Value, root, tagID, nil); err != nil {
			return nil, err
		}
	}
	p.normalize(root.TasksNode)
	if len(root.Tasks) > 0 {
		workflow.TasksNode = root.TasksNode
	} else {
		workflow.TasksNode = &TasksNode{
			Tasks: []*Task{root},
		}
	}
	p.workflow = workflow
	return workflow, nil

}

func (p *InlineWorkflow) normalize(node *TasksNode) {
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

func (p *InlineWorkflow) buildTask(name string, source interface{}) *Task {
	var task = &Task{}
	if toolbox.IsSlice(source) && toolbox.IsMap(source) {
		_ = toolbox.DefaultConverter.AssignConverted(task, source)
	}
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

func getTemplateNode(source interface{}) *Template {
	if source == nil || !(toolbox.IsSlice(source) || toolbox.IsMap(source)) {
		return nil
	}
	var template = &Template{}
	toolbox.DefaultConverter.AssignConverted(template, source)
	if len(template.Template) == 0 || template.Range == "" {
		return nil
	}
	return template
}

func (p *InlineWorkflow) buildAction(name string, actionAttributes, actionRequest map[string]interface{}, tagId string) (*Action, error) {
	var result = &Action{
		AbstractNode:   &AbstractNode{},
		ServiceRequest: &ServiceRequest{},
		Repeater:       &Repeater{},
	}

	util.Append(actionRequest, p.Defaults, false)

	if action, ok := actionAttributes["action"]; ok {
		actionAttributes["request"], _ = util.NormalizeMap(actionRequest, false)
		selector := ActionSelector(toolbox.AsString(action))
		actionAttributes["service"] = selector.Service()
		actionAttributes["action"] = selector.Action()
	} else {
		workflow := toolbox.AsString(actionAttributes["workflow"])
		actionAttributes["action"] = "run"
		selector := WorkflowSelector(workflow)
		actionAttributes["request"] = map[string]interface{}{
			"params": actionRequest,
			"tasks":  selector.Tasks(),
			"URL":    selector.URL(),
		}
	}
	if err := toolbox.DefaultConverter.AssignConverted(result, actionAttributes); err != nil {
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

func (p *InlineWorkflow) hasActionNode(source interface{}) bool {
	if source == nil {
		return false
	}
	var result = false
	attributes, _ := util.NormalizeMap(source, false)

	if isActionNode(attributes) {
		return true
	}

	_ = toolbox.ProcessMap(attributes, func(key, value interface{}) bool {
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

func (p *InlineWorkflow) buildWorkflowNodes(name string, source interface{}, parentTask *Task, tagID string, state data.Map) error {
	if state != nil {
		source = state.Expand(source)
	}
	actionAttributes, actionRequest, err := p.split(source, state)
	if err != nil {
		return err
	}
	var task *Task
	isTemplateNode := false
	if parentTask != nil {
		template := getTemplateNode(source)
		if template != nil {
			task = p.buildTask(name, source)
			parentTask.Tasks = append(parentTask.Tasks, task)
			isTemplateNode = true
			if err = template.Expand(task, name, p); err != nil {
				return err
			}
		}
	}

	if isActionNode(actionAttributes) {
		action, err := p.buildAction(name, actionAttributes, actionRequest, tagID)
		if err != nil {
			return err
		}
		task := parentTask
		if !parentTask.multiAction {
			task = p.buildTask(name, map[string]interface{}{})
			parentTask.Tasks = append(parentTask.Tasks, task)
		}

		task.Actions = append(task.Actions, action)
		return nil
	}

	if !p.hasActionNode(actionAttributes) {
		return nil
	}

	if !isTemplateNode {
		task = p.buildTask(name, source)
		parentTask.Tasks = append(parentTask.Tasks, task)
	}

	var nodeAttributes = make(map[string]interface{})
	var buildErr error

	if err := toolbox.ProcessMap(source, func(key, value interface{}) bool {
		textKey := strings.ToLower(toolbox.AsString(key))
		if isTemplateNode && "template" == textKey {
			return true
		}
		if textKey == "logging" || textKey == "when" { //abstract node attributes
			nodeAttributes[textKey] = value
		}
		flagAsMultiActionIfMatched(textKey, task, value)
		if !toolbox.IsSlice(value) {
			return true
		}
		buildErr = p.buildWorkflowNodes(toolbox.AsString(key), value, task, tagID+"_"+task.Name, state)
		if buildErr != nil {
			return false
		}
		nodeAttributes[textKey] = value
		return true
	}); err != nil {
		return err
	}

	if task == nil {
		task = parentTask
	}
	if _, actionNode := nodeAttributes["action"]; !actionNode && !isTemplateNode {

		if taskAttributes, _, err := p.split(nodeAttributes, state); err == nil {
			if len(taskAttributes) > 0 {
				tempTask := &Task{}
				if err = toolbox.DefaultConverter.AssignConverted(&tempTask, taskAttributes); err == nil {
					if tempTask.AbstractNode != nil {
						task.Init = tempTask.Init
						task.Post = tempTask.Post
						task.When = tempTask.When
						task.Logging = tempTask.Logging
					}
				}
			}
		}
	}

	return buildErr

}

func flagAsMultiActionIfMatched(textKey string, task *Task, value interface{}) {
	for _, key := range multiActionKeys {
		if textKey == key && toolbox.IsBool(value) {
			task.multiAction = toolbox.AsBoolean(value)
			break
		}
	}
}
