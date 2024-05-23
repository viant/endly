package project

import (
	"encoding/json"
	"fmt"
	"github.com/viant/endly/model/graph/yml"
	"gopkg.in/yaml.v3"
	"path"
	"sort"
	"strings"
)

type (
	Workflow struct {
		ID            string      `json:"SessionID,omitempty"`
		Position      int         `json:"POSITION,omitempty"`
		ParentID      string      `json:"PARENT_ID,omitempty"`
		Revision      string      `json:"REVISION,omitempty"`
		URI           string      `json:"URI,omitempty"`
		ProjectID     string      `json:"PROJECT_ID,omitempty"`
		Name          string      `json:"NAME,omitempty"`
		Description   string      `json:"DESCRIPTION,omitempty"`
		Init          string      `jsonx:"inline" json:"INIT,omitempty"`
		Post          string      `jsonx:"inline" json:"POST,omitempty"`
		Steps         []*Task     `json:"-"`
		Assets        []*Asset    `json:"-"`
		Workflows     []*Workflow `json:"-"`
		Template      string      `json:"TEMPLATE,omitempty"`
		InstanceIndex int         `json:"INSTANCE_INDEX,omitempty"`
		InstanceTag   string      `json:"INSTANCE_TAG,omitempty"`
	}

	//Revision represents a workflow revision
	Revision struct {
		//SessionID represents revision ID
		ID string
		//WorkflowID represents workflow ID
		Principal string
		//Comment represents revision comment
		Comment string
		//Diff represents revision diff
		Diff string
	}
)

func (w *Workflow) FileName() string {
	if path.Ext(w.URI) != "" {
		return w.URI
	}
	return w.URI + ".yaml"
}

func (w *Workflow) MarshalYAML() (interface{}, error) {
	workflow := (*yml.Node)(yml.NewMap())
	if w.Init != "" {
		v, err := w.marshalVariables(w.Init)
		if err != nil {
			return nil, err
		}
		workflow.Put("init", v)
	}
	holder := (*yml.Node)(yml.NewMap())
	sort.Slice(w.Steps, func(i, j int) bool {
		return w.Steps[i].Position < w.Steps[j].Position
	})
	err := w.marshalTasks(holder, "")
	if err != nil {
		return nil, err
	}
	workflow.Put("pipeline", (*yaml.Node)(holder))
	if w.Post != "" {
		var v interface{}
		v, err := w.marshalVariables(w.Post)
		if err != nil {
			return nil, err
		}
		workflow.Put("post", v)
	}
	return (*yaml.Node)(workflow), nil
}

func (w *Workflow) marshalVariables(vars string) (interface{}, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(vars), &v); err != nil {
		return nil, err
	}
	aSlice := (*yml.Node)(yml.NewMap())
	switch actual := v.(type) {
	case []interface{}:
		for _, item := range actual {
			decl, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("unsupported variable declaration type: %T", item)
			}
			value := w.adjustMarshalledVariableValue(decl)
			aSlice.Put(decl["name"].(string), value)
		}
	default:
		return nil, fmt.Errorf("unsupported variable type: %T", actual)
	}
	return (*yaml.Node)(aSlice), nil
}

func (w *Workflow) adjustMarshalledVariableValue(decl map[string]interface{}) interface{} {
	value := decl["value"]
	if text, ok := value.(string); ok {
		if text == "null" {
			value = nil
		}
		text = strings.Trim(text, "'")
		if strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}") {
			var v interface{}
			if err := json.Unmarshal([]byte(text), &v); err == nil {
				return v
			}
		}
		if strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]") {
			var v []interface{}
			if err := json.Unmarshal([]byte(text), &v); err == nil {
				return v
			}
		}
	}
	return value
}

func (w *Workflow) marshalTasks(tasks *yml.Node, parentId string) error {
	for _, step := range w.Steps {
		if step.ParentId != parentId {
			continue
		}
		task, err := step.MarshalYAML()
		if err != nil {
			return fmt.Errorf("failed to marshal node: %s, %w", step.Tag, err)
		}
		taskNode := task.(*yaml.Node)
		if step.SubPath != "" {
			tmpl := (*yml.Node)(taskNode).Lookup("template")
			taskNode = (*yaml.Node)(tmpl)
		}
		if err := w.marshalTasks((*yml.Node)(taskNode), step.ID); err != nil {
			return err
		}
		tasks.Put(step.Tag, task)
	}
	return nil
}
