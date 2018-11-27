package model

import (
	"fmt"
	"github.com/viant/endly/util"
	"github.com/viant/neatly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"strings"
)

//Template represents inline workflow template to dynamically Expand actions - idea borrowed from neatly format: https://github.com/viant/neatly/
type Template struct {
	SubPath     string            `description:"sub path for dynamic resource template expansion: i.e. use_cases/${index}*"`
	Tag         string            `description:"grouping tag i.e Test"`
	Range       string            `description:"range expression i.e 2..003  where upper bound number drives padding $index variable"`
	Description string            `description:"reference to file containing tagDescription i.e. @use_case,  file reference has to start with @"`
	Data        map[string]string `description:"map of data references, where key is workflow.data target, and value is a file within expanded dynamically subpath or workflow path fallback. Value has to start with @"`
	inline      *InlineWorkflow
	Template    []interface{}
}

func (t *Template) Expand(task *Task, parentTag string, inline *InlineWorkflow) error {
	if t.Tag == "" {
		if t.Tag = task.Name; t.Tag == "" {
			t.Tag = parentTag
		}
	}
	t.inline = inline
	tag := buildTag(t, inline)
	task.multiAction = true
	iterator := tag.Iterator
	var workflowData = data.Map(t.inline.Data)

	for tag.HasActiveIterator() {
		tempTask := NewTask(task.Name, true)
		index := iterator.Index()
		state := t.buildTagState(index, tag)
		tagPath := state.GetString("path")
		t.inline.tagPathURL = tagPath
		if len(t.Data) > 0 {
			if err := t.loadWorkflowData(tagPath, workflowData, state); err != nil {
				return fmt.Errorf("failed to load data: %v", err)
			}
		}
		var err error
		toolbox.ProcessMap(t.Template, func(key, value interface{}) bool {
			if err = inline.buildWorkflowNodes(toolbox.AsString(key), value, tempTask, t.Tag, state); err != nil {
				return false
			}
			return true
		})
		if err != nil {
			return err
		}
		description := ""
		if t.Description != "" {
			util.LoadResource(tagPath, t.Description, &description)
		}
		actions := flattenAction(tempTask, tempTask, tag, description)
		task.Actions = append(task.Actions, actions...)
		if !iterator.Next() {
			break
		}
	}
	t.inline.tagPathURL = ""
	return nil
}

func (t *Template) loadWorkflowData(tagPath string, workflowData data.Map, state data.Map) error {
	var baseURLs = []string{tagPath, t.inline.baseURL}
	var err error
	for k, v := range t.Data {
		k = state.ExpandAsText(k)
		hasWildCard := strings.Contains(v, "*")
		var resourceURLs = make([]string, 0)
		if hasWildCard {
			resourceURLs, err = util.ListResource(baseURLs, v)
			if util.IsNotSuchResourceError(err) {
				continue
			}
			if err != nil {
				return err
			}
		}
		if len(resourceURLs) > 0 {
			for _, resourceURL := range resourceURLs {
				base, URI := toolbox.URLSplit(resourceURL)
				loaded, err := util.LoadData([]string{base}, "@"+URI)
				if err != nil {
					return err
				}
				addLoadedData(loaded, state, k, workflowData)
			}
			continue
		}
		loaded, err := util.LoadData(baseURLs, v)
		if util.IsNotSuchResourceError(err) {
			continue
		}
		if err != nil {
			return err
		}
		addLoadedData(loaded, state, k, workflowData)
	}
	return nil
}

func addLoadedData(loaded interface{}, state data.Map, k string, workflowData data.Map) {
	loaded = state.Expand(loaded)
	collectionSignatureCount := strings.Count(k, "[]")
	if collectionSignatureCount > 0 {
		k = strings.Replace(k, "[]", "", collectionSignatureCount)
		var collection *data.Collection
		collectionValue, ok := workflowData.GetValue(k)
		if !ok {
			collection = data.NewCollection()
			workflowData.SetValue(k, collection)
		} else {
			collection, _ = collectionValue.(*data.Collection)
		}
		if collection == nil {
			collection = data.NewCollection()
			workflowData.SetValue(k, collection)
		}
		if toolbox.IsSlice(loaded) {
			for _, item := range toolbox.AsSlice(loaded) {
				collection.Push(item)
			}
		} else {
			collection.Push(loaded)
		}
	} else {
		workflowData.SetValue(k, loaded)
	}
}

func (t *Template) buildTagState(index string, tag *neatly.Tag) data.Map {
	var state = data.NewMap()
	state.Put("index", index)
	if t.SubPath != "" {
		tag.SetSubPath(state.ExpandAsText(t.SubPath))
	}
	tagPath := toolbox.URLPathJoin(t.inline.baseURL, tag.Subpath)
	state.Put("subpath", tag.Subpath)
	state.Put("tagId", tag.TagID())
	state.Put("subPath", tag.Subpath)
	state.Put("path", tagPath)
	return state
}

func flattenAction(parent *Task, task *Task, tag *neatly.Tag, description string) []*Action {
	var result = make([]*Action, 0)
	isRootTask := parent == task
	if !isRootTask {
		tag.Group = parent.Name
	}
	if len(task.Actions) > 0 {
		result = task.Actions
		for i := range result {
			action := result[i]
			action.TagID = tag.TagID()
			action.TagIndex = tag.Iterator.Index()
			action.Tag = tag.Name
			if i == 0 {
				action.TagDescription = description
			}
		}
	}
	if task.TasksNode != nil && len(task.Tasks) > 0 {
		for _, subTask := range task.Tasks {
			actions := flattenAction(task, subTask, tag, description)
			result = append(result, actions...)
		}
	}
	return result
}

func buildTag(t *Template, inline *InlineWorkflow) *neatly.Tag {
	key := t.Tag + "{" + t.Range + "}"
	ownerURL := toolbox.URLPathJoin(inline.baseURL, inline.name+".yaml")
	tag := neatly.NewTag(inline.name, url.NewResource(ownerURL), key, 0)
	return tag
}
