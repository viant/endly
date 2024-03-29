package graph

import (
	"encoding/json"
	"fmt"
	"github.com/viant/endly/model/yml"
	"github.com/viant/toolbox"
	"gopkg.in/yaml.v3"
	"strings"
)

const (
	TypeWorkflow = Type(iota)
	TypeVariable
	TypeTask
	TypeAction
	TypeTemplate
)

type (
	Type int

	Node struct {
		*yml.Node
		Type Type
		Name string
		IsTemplate bool
	}
)

var reservcedKeys = map[string]bool{
	"init":        true,
	"post":        true,
	"variables":   true,
	"when":        true,
	"sleeptimems": true,
	"logging":     true,
	"description": true,
	"input":       true,
	"repeat":      true,
	"thinktimems": true,
	"exit":        true,
	"action":      true,
	"service":     true,
	"extract":     true,
}

func (n *Node) Value() interface{} {
	switch n.Kind {
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str":
			return n.Node.Value
		case "!!bool":
			return toolbox.AsBoolean(n.Node.Value)
		case "!!nil":
			return nil
		case "!!float":
			return toolbox.AsFloat(n.Node.Value)
		case "!!int":
			return toolbox.AsInt(n.Node.Value)
		default:
			return n.Node.Value
		}
	case yaml.MappingNode:
		var aMap = make(map[string]interface{})
		for i := 0; i < len(n.Content); i += 2 {
			key := n.Content[i].Value
			value := &Node{Node: (*yml.Node)(n.Content[i+1])}
			aMap[key] = value.Value()
		}
		return aMap
	case yaml.SequenceNode:
		var aSlice []interface{}
		for i := 0; i < len(n.Content); i++ {
			value := &Node{Node: (*yml.Node)(n.Content[i+1])}
			aSlice = append(aSlice, value.Value())
		}
		return aSlice
	}
	return nil
}

func (n *Node) HasInit() {
	init := n.Node.Lookup("init")
	if init == nil {
		return
	}
}

func (n *Node) Task(name string) (*Node, error) {
	var holder *yml.Node
	switch n.Type {
	case TypeWorkflow:
		holder = n.Node.Lookup("pipeline")
		if holder == nil {
			return nil, nil
		}
	case TypeTask, TypeTemplate:
		holder = n.Node
	}
	ret := holder.Lookup(name)
	if ret == nil {
		return nil, nil
	}
	task := &Node{Name: name, Node: ret, Type: TypeAction}
	if action := task.Lookup("action"); action != nil {
		task.Type = TypeAction
	}
	return task, nil
}

func (n *Node) IsTaskNode() bool {
	result := false
	_ = n.Tasks(func(name string, node *Node) error {
		switch node.Type {
		case TypeTask:
			if isTask := node.IsTaskNode(); isTask {
				result = isTask
			}
		case TypeAction:
			result = true
		}
		return nil
	})
	return result
}

func (n *Node) Tasks(callback func(name string, node *Node) error) error {
	switch n.Type {
	case TypeWorkflow:
		pipeline := n.Node.Lookup("pipeline")
		if pipeline == nil {
			return nil
		}
		return pipeline.Pairs(func(key string, node *yml.Node) error {
			taskNode := &Node{Name: key, Node: node, Type: TypeTask}
			if action := node.Lookup("action"); action != nil {
				taskNode.Type = TypeAction
			}
			return callback(key, taskNode)
		})
	case TypeTask:
		return n.Pairs(func(key string, node *yml.Node) error {
			taskNode := &Node{Name: key, Node: node, Type: TypeAction}
			if action := node.Lookup("action"); action != nil {
				taskNode.Type = TypeAction
			}
			return callback(key, taskNode)
		})
	case TypeTemplate:
		return n.Pairs(func(key string, node *yml.Node) error {
			taskNode := &Node{Name: key, Node: node, Type: TypeTask, IsTemplate: true}
			if action := node.Lookup("action"); action != nil {
				taskNode.Type = TypeAction
			}
			return callback(key, taskNode)
		})
	}
	return nil
}

func (n *Node) Variables(ns string) (string, error) {
	vars := n.Node.Lookup(ns)
	if vars == nil {
		return "", nil
	}
	if n.IsTaskNode() {
		return "", nil
	}

	var result []string
	err := n.variables(func(name string, node *Node) error {
		value := ""
		switch actual := node.Value().(type) {
		case map[string]interface{}, []interface{}:
			enc, err := json.Marshal(actual)
			if err != nil {
				return err
			}
			value = string(enc)
		case string:
			value = actual
		default:
			value = toolbox.AsString(actual)
		}

		variable := name + ": " + value
		result = append(result, variable)
		return nil
	}, vars)
	data, err := json.Marshal(result)
	return string(data), err
}

func (n *Node) Extracts(callback func(name string, node *Node) error) error {
	init := n.Node.Lookup("extract")
	if init == nil {
		return nil
	}
	return n.variables(callback, init)
}

func (n *Node) variables(callback func(name string, node *Node) error, init *yml.Node) error {
	switch init.Kind {
	case yaml.MappingNode:
		return init.Pairs(func(key string, node *yml.Node) error {
			return callback(key, &Node{Node: node, Type: TypeVariable})
		})
	case yaml.SequenceNode:
		return init.Items(func(index int, node *yml.Node) error {
			if len(node.Content) == 0 {
				return nil
			}
			name := node.Content[0].Value
			valueNode := (*yml.Node)(node.Content[1])
			return callback(name, &Node{Node: valueNode, Type: TypeVariable})
		})
	}
	return nil
}

func (n *Node) WorkflowMap() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	err := n.abstractNode(result)
	return result, err
}

func (n *Node) Request() (interface{}, error) {
	input := n.Node.Lookup("input")
	if input != nil {
		valueNode := &Node{Node: input}
		return valueNode.Value(), nil
	}
	var result = make(map[string]interface{})
	err := n.Pairs(func(key string, node *yml.Node) error {
		if strings.HasPrefix(key, ":") { //explicit action attribute
			result[key[1:]] = node.Value
			return nil
		}
		if strings.HasPrefix(key, "@") { //explicit action attribute
			return nil
		}
		if reservcedKeys[strings.ToLower(key)] {
			return nil
		}
		result[key] = node.Value
		return nil
	})
	return result, err
}

func (n *Node) String(name string) (interface{}, bool) {
	value, ok := n.Scalar(name)
	if !ok {
		return nil, false
	}
	return value.(string), true
}

func (n *Node) Scalar(name string) (interface{}, bool) {
	match := n.Node.Lookup(name)
	if match == nil {
		return nil, false
	}
	node := &Node{Node: match}
	return node.Value(), true
}

// ActionMap returns action map
func (n *Node) ActionMap() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	err := n.abstractNode(result)
	if err != nil {
		return nil, err
	}
	err = n.Pairs(func(key string, node *yml.Node) error {
		switch strings.ToLower(key) {
		case "repeat", "thinktimems", "exit", "action", "service", "async", "skip":
			result[key] = node.Value
		}
		return nil
	})
	return result, err
}

func (n *Node) Data() string {
	data := n.Node.Lookup("data")
	if data == nil {
		return ""
	}
	result := make(map[string]string)
	_ = data.Pairs(func(key string, node *yml.Node) error {
		result[key] = node.Value
		return nil
	})
	jsonData, _ := json.Marshal(result)
	return string(jsonData)
}

// TaskMap returns task map
func (n *Node) TaskMap() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	err := n.abstractNode(result)
	if err != nil {
		return nil, err
	}
	err = n.Pairs(func(key string, node *yml.Node) error {
		switch strings.ToLower(key) {
		case "defer", "onerror", "fail", "subpath", "tag", "range":
			result[key] = node.Value
		}
		return nil
	})
	return result, err
}

func (n *Node) Template() *Node {
	template := n.Node.Lookup("template")
	if template == nil {
		return nil
	}
	return &Node{Node: template, Type: TypeTemplate}
}

func (n *Node) abstractNode(result map[string]interface{}) error {
	switch n.Type {
	case TypeWorkflow, TypeAction, TypeTask:
	default:
		return fmt.Errorf("invalid node type: %v", n.Type)
	}
	if n.Name != "" {
		result["name"] = n.Name
	}
	err := n.Pairs(func(key string, node *yml.Node) error {
		switch strings.ToLower(key) {
		case "when", "sleeptimems", "logging", "description", "comments":
			result[key] = node.Value
		}
		return nil
	})
	return err
}

func NewWorkflowNode(name string, node *yaml.Node) *Node {
	n := (*yml.Node)(node)
	return &Node{Name: name, Node: n, Type: TypeWorkflow}
}
