package model

import (
	"context"
	"fmt"
	"github.com/viant/afs"
	"github.com/viant/afs/file"
	"github.com/viant/afs/storage"
	url "github.com/viant/afs/url"
	"github.com/viant/endly/internal/util"
	"github.com/viant/endly/model/graph"
	"github.com/viant/endly/model/location"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"path"
	"strings"
)

type Template struct {
	SubPath     string            `description:"sub path for dynamic resource template expansion: i.e. use_cases/${index}*"`
	Tag         string            `description:"grouping tag i.e Test"`
	Range       string            `description:"range expression i.e 2..003  where upper bound number drives padding $index variable"`
	Description string            `description:"reference to file containing tagDescription i.e. @use_case,  file reference has to start with @"`
	Data        map[string]string `description:"map of data references, where key is workflow.data target, and value is a file within expanded dynamically subpath or workflow path fallback. Value has to start with @"`
	Tasks       []*Task           `description:"tasks to expand"`
}

// TransientTemplate represents inline workflow template to dynamically Expand actions - idea borrowed from neatly format: https://github.com/viant/neatly/
type TransientTemplate struct {
	SubPath     string            `description:"sub path for dynamic resource template expansion: i.e. use_cases/${index}*"`
	Tag         string            `description:"grouping tag i.e Test"`
	Range       string            `description:"range expression i.e 2..003  where upper bound number drives padding $index variable"`
	Description string            `description:"reference to file containing tagDescription i.e. @use_case,  file reference has to start with @"`
	Data        map[string]string `description:"map of data references, where key is workflow.data target, and value is a file within expanded dynamically subpath or workflow path fallback. Value has to start with @"`
	Template    []interface{}     `description:"template to expand"`
	inline      *Inlined
}

func (t *TransientTemplate) Expand(task *Task, parentTag string, inline *Inlined) error {
	if t.Tag == "" {
		t.Tag = "$pathMatch"
		//if t.Tag = task.Name; t.Tag == "" {
		//	t.Tag = parentTag
		//}
	}

	fs := afs.New()

	t.inline = inline
	var instances *graph.Instances

	if t.SubPath != "" {
		templateURL := url.Join(t.inline.baseURL, t.SubPath)
		parent, name := url.Split(templateURL, file.Scheme)
		holder, err := fs.Object(context.Background(), parent)
		if err != nil {
			err = fmt.Errorf("failed to LookupValueNode parent: %v, %v", parent, err)
		}
		objects, err := fs.List(context.Background(), parent)
		if err != nil {
			return err
		}
		instances = graph.NewInstances(holder.URL(), name, objects)
		if t.Range == "" {
			t.Range = instances.Range()
		}
	}

	tag := buildTag(t, inline)
	task.multiAction = true
	iterator := tag.Iterator
	var workflowData = data.Map(t.inline.Data)
	for tag.HasActiveIterator() {
		tempTask := NewTask(task.Name, true)
		tag.Group = task.Name
		index := iterator.Index()
		state := t.buildTagState(index, tag, instances)
		tagPath := state.GetString("path")
		t.inline.tagPathURL = tagPath
		if len(t.Data) > 0 {
			if err := t.loadWorkflowData(tagPath, workflowData, state); err != nil {
				return fmt.Errorf("failed to load data: %v", err)
			}
		}
		var err error
		_ = toolbox.ProcessMap(t.Template, func(key, value interface{}) bool {
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
			_, _ = util.LoadResource(tagPath, t.Description, &description)
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

func (t *TransientTemplate) loadWorkflowData(tagPath string, workflowData data.Map, state data.Map) error {

	var baseURLs = []string{tagPath, toolbox.URLPathJoin(t.inline.baseURL, "default"), t.inline.baseURL}
	var err error

	for k, loc := range t.Data {
		k = state.ExpandAsText(k)
		hasWildCard := strings.Contains(loc, "*")
		var resourceURLs = make([]string, 0)
		if hasWildCard {
			resourceURLs, err = util.ListResource(baseURLs, loc)
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
		URI := strings.Replace(loc, "@", "", 1)
		fs := afs.New()
		object, _ := fs.Object(context.Background(), url.Join(baseURLs[0], URI))
		if object != nil && object.IsDir() {
			loaded, err := loadKeyedAssets(fs, object, baseURLs, state)
			if err != nil {
				return err
			}
			addLoadedMapData(loaded, state, k, workflowData)
			continue
		}
		loaded, err := util.LoadData(baseURLs, loc)
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

func loadKeyedAssets(fs afs.Service, object storage.Object, baseURLs []string, state data.Map) (map[string]interface{}, error) {
	items, err := fs.List(context.Background(), object.URL())
	if err != nil {
		return nil, err
	}
	var assets = map[string]interface{}{}
	var URLs = make([]string, 3)
	copy(URLs, baseURLs)
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		URLs[0] = item.URL()
		key := item.Name()
		if index := strings.LastIndex(key, "."); index != -1 {
			key = key[:index]
		}
		baseURL, URI := url.Split(item.URL(), file.Scheme)
		var loaded interface{}
		if _, err := util.LoadResource(baseURL, "@"+URI, &loaded); err != nil {
			return nil, err
		}
		if err != nil {
			return nil, err
		}
		assets[key] = loaded
	}
	return assets, nil
}

func addLoadedMapData(loaded map[string]interface{}, state data.Map, stateKey string, workflowData data.Map) {
	collectionSignatureCount := strings.Count(stateKey, "[]")
	if collectionSignatureCount > 0 {
		stateKey = strings.Replace(stateKey, "[]", "", collectionSignatureCount)
	}

	for key, value := range loaded {
		k := stateKey + "." + key
		value = state.Expand(value)
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
		if toolbox.IsSlice(value) {
			for _, item := range toolbox.AsSlice(value) {
				collection.Push(item)
			}
		} else {
			collection.Push(value)
		}
	}
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

func (t *TransientTemplate) buildTagState(index string, tag *Tag, instances *graph.Instances) data.Map {
	var state = data.NewMap()
	state.Put("index", index)
	if t.SubPath != "" {
		instance := instances.Lookup(toolbox.AsInt(index))
		if index := strings.LastIndex(t.SubPath, "/"); index != -1 {
			parent, _ := path.Split(t.SubPath)
			tag.Subpath = path.Join(parent, instance.Object.Name())

		} else {
			tag.Subpath = instance.Object.Name()
		}
		tag.PathMatch = instance.Tag
		state.Put("tag", instance.Tag)
	}

	tagPathURL := toolbox.URLPathJoin(t.inline.baseURL, tag.Subpath)
	state.Put("subpath", tag.Subpath)
	state.Put("tagId", tag.TagID())
	state.Put("subPath", tag.Subpath)
	state.Put("pathMatch", tag.PathMatch)
	state.Put("URL", tagPathURL)
	state.Put("path", location.NewResource(tagPathURL).Path())

	return state
}

func flattenAction(parent *Task, task *Task, tag *Tag, description string) []*Action {
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

			action.Tag = tag.Expand(tag.Name)
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

func buildTag(t *TransientTemplate, inline *Inlined) *Tag {
	key := t.Tag + "{" + t.Range + "}"
	ownerURL := toolbox.URLPathJoin(inline.baseURL, inline.name+".yaml")
	tag := NewTag(inline.name, location.NewResource(ownerURL), key, 0)
	return tag
}
