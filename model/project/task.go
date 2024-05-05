package project

import (
	"encoding/json"
	"github.com/viant/endly/model/graph/yml"
	"gopkg.in/yaml.v3"
	"strings"
)

type (
	//Task represents a task
	Task struct {
		ID            string   `json:"SessionID,omitempty" yaml:"-"`
		WorkflowID    string   `json:"WORKFLOW_ID,omitempty" yaml:"-"`
		ParentId      string   `json:"PARENT_ID,omitempty" yaml:"-"`
		Position      int      `json:"POSITION,omitempty" yaml:"-"`
		Tag           string   `json:"TAG,omitempty" yaml:"-"`
		Init          string   `jsonx:"inline" json:"INIT,omitempty" yaml:"-"`
		Post          string   `jsonx:"inline" json:"POST,omitempty" yaml:"-"`
		Description   string   `json:"DESCRIPTION,omitempty" yaml:",omitempty"`
		When          string   `sqlx:"WHEN_EXPR" json:"WHEN_EXPR,omitempty" yaml:"when,omitempty"`
		Exit          string   `sqlx:"EXIT_EXPR" json:"EXIT_EXPR,omitempty" yaml:"exit,omitempty"`
		OnError       string   `json:"ON_ERROR,omitempty" yaml:"onError,omitempty"`
		Deferred      string   `json:"DEFERRED,omitempty" yaml:",omitempty"`
		Service       string   `json:"SERVICE,omitempty" yaml:",omitempty"`
		Action        string   `json:"ACTION,omitempty" yaml:",omitempty"`
		Input         string   `json:"INPUT,omitempty" yaml:"-"`
		InputURI      string   `json:"INPUT_URI,omitempty" yaml:"uri,omitempty"`
		Async         bool     `json:"ASYNC,omitempty" yaml:",omitempty"`
		Skip          string   `sqlx:"SKIP_EXPR" json:"SKIP_EXPR,omitempty" yaml:",omitempty"`
		Fail          bool     `json:"FAIL,omitempty" yaml:",omitempty"`
		IsTemplate    bool     `json:"IS_TEMPLATE,omitempty" yaml:"-"`
		SubPath       string   `json:"SUB_PATH,omitempty"  yaml:",omitempty"`
		Range         string   `sqlx:"RANGE_EXPR" json:"RANGE_EXPR,omitempty" yaml:",omitempty"`
		Data          string   `jsonx:"inline" json:"DATA,omitempty" yaml:"-"`
		Variables     string   `jsonx:"inline" json:"VARIABLES,omitempty" yaml:"-"`
		Extracts      Extracts `jsonx:"inline" json:"EXTRACTS,omitempty" yaml:",omitempty"`
		SleepTimeMs   int      `json:"SLEEP_TIME_MS,omitempty" yaml:"sleepTimeMs,omitempty"`
		ThinkTimeMs   int      `json:"THINK_TIME_MS,omitempty"  yaml:"thinkTimeMs,omitempty"`
		Logging       *bool    `json:"LOGGING,omitempty"  yaml:",omitempty"`
		Repeat        int      `sqlx:"REPEAT_RUN" json:"REPEAT_RUN,omitempty"  yaml:",omitempty"`
		InstanceIndex int      `json:"INSTANCE_INDEX,omitempty"  yaml:"-"`
		InstanceTag   string   `json:"INSTANCE_TAG,omitempty" yaml:"-"`
	}

	//Extract represents a data extraction rule
	Extract struct {
		RegExpr  string `description:"regular expression with oval bracket to extract match pattern" yaml:",omitempty" `           //regular expression
		Key      string `description:"state key to store a match" yaml:",omitempty"`                                               //state key to store a match
		Reset    bool   `description:"reset the key in the context before evaluating this data extraction rule" yaml:",omitempty"` //reset the key in the context before evaluating this data extraction rule
		Required bool   `description:"require that at least one pattern match is returned" yaml:",omitempty"`                      //require that at least one pattern match is returned
	}

	//Extracts represents a list of data extraction rules
	Extracts []*Extract
)

// MarshalYAML marshals task to yaml
func (t *Task) MarshalYAML() (interface{}, error) {
	type task Task
	clone := (*task)(t)
	if clone.Service == "workflow" {
		clone.Service = ""
	}
	if clone.Service != "" {
		clone.Action = clone.Service + ":" + clone.Action
	}
	orig := &yaml.Node{}
	if err := orig.Encode(clone); err != nil {
		return nil, err
	}
	orig.Style = 0

	if t.Input != "" {
		err := t.marshalStructured(orig, "input", t.Input)
		if err != nil {
			return nil, err
		}
	}

	if t.Variables != "" {
		err := t.marshalStructured(orig, "variables", t.Variables)
		if err != nil {
			return nil, err
		}
	}

	if t.Data != "" {
		err := t.marshalStructured(orig, "data", t.Data)
		if err != nil {
			return nil, err
		}
	}

	if t.Post != "" {
		err := t.marshalStructured(orig, "post", t.Post)
		if err != nil {
			return nil, err
		}
	}
	if t.SubPath != "" {
		(*yml.Node)(orig).Put("template", yml.NewMap())
	}
	return orig, nil
}

func (t *Task) marshalStructured(orig *yaml.Node, key string, literal string) error {
	var value interface{}
	if strings.HasPrefix(literal, "'") && strings.HasSuffix(literal, "'") {
		literal = literal[1 : len(literal)-1]
	}
	if err := json.Unmarshal([]byte(literal), &value); err != nil {
		return err
	}
	switch actual := value.(type) {
	case map[string]interface{}:
		for k, v := range actual {
			(*yml.Node)(orig).Put(k, v)
		}
	default:
		(*yml.Node)(orig).Put(key, value)

	}
	return nil
}

// Method returns task method
func (t *Task) Method() string {
	if t.Action != "" {
		return t.Service + ":" + t.Action
	}
	return ""
}

func (t *Task) IsWorkflowRun() bool {
	return t.Method() == "workflow:run"

}

// GetData returns task data
func (t *Task) GetData() map[string]string {
	data := make(map[string]string)
	if err := json.Unmarshal([]byte(t.Data), &data); err != nil {
		return nil
	}
	return data
}

// SetID sets task id
func (t *Task) SetID(prefix, name string) {
	t.ID = prefix + "/" + name
	t.Tag = name
}
