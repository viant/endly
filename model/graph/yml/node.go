package yml

import (
	"fmt"
	"github.com/viant/toolbox"
	"gopkg.in/yaml.v3"
	"reflect"
)

type (
	Node yaml.Node
)

func (n *Node) Lookup(name string) *Node {
	return (*Node)(Nodes(n.Content).LookupValueNode(name))
}

func (n *Node) Items(callback func(index int, node *Node) error) error {
	for i := 0; i < len(n.Content); i++ {
		value := n.Content[i]
		nodeValue := (*Node)(value)
		if err := callback(i, nodeValue); err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) Pairs(callback func(key string, node *Node) error) error {
	for i := 0; i < len(n.Content); i += 2 {
		key := n.Content[i].Value
		value := n.Content[i+1]
		nodeValue := (*Node)(value)
		if err := callback(key, nodeValue); err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) Interface() interface{} {
	switch n.Kind {
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str":
			return n.Value
		case "!!bool":
			return toolbox.AsBoolean(n.Value)
		case "!!nil":
			return nil
		case "!!float":
			return toolbox.AsFloat(n.Value)
		case "!!int":
			return toolbox.AsInt(n.Value)
		default:
			return n.Value
		}
	case yaml.MappingNode:
		var aMap = make(map[string]interface{})
		for i := 0; i < len(n.Content); i += 2 {
			key := n.Content[i].Value
			value := (*Node)(n.Content[i+1])
			aMap[key] = value.Interface()
		}
		return aMap
	case yaml.SequenceNode:
		var aSlice = make([]interface{}, 0)
		for i := 0; i < len(n.Content); i++ {
			value := (*Node)(n.Content[i])
			aSlice = append(aSlice, value.Interface())
		}
		return aSlice
	}
	return nil
}

func (n *Node) Append(value interface{}) {
	switch n.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
	default:
		panic("not a sequence node")
	}
	n.Content = append(n.Content, ValueNode(value))
}

func (n *Node) Put(key string, value interface{}) {
	if n.Kind != yaml.MappingNode { //sanity check
		panic("not a map node")
	}

	n.Content = append(n.Content, newScalar(key))
	n.Content = append(n.Content, ValueNode(value))
}

func ValueNode(value interface{}) *yaml.Node {
	if value == nil {
		return newScalar(nil)
	}
	switch actual := value.(type) {
	case *Node:
		return (*yaml.Node)(actual)
	case yaml.Node:
		return &actual
	case *yaml.Node:
		return actual
	case Node:
		n := &actual
		return (*yaml.Node)(n)
	case string, []byte, int, int64, uint64, float64, float32, bool:
		return newScalar(value)
	case map[string]interface{}:
		aMap := (*Node)(NewMap())
		for k, v := range actual {
			aMap.Put(k, v)
		}
		return (*yaml.Node)(aMap)
	case map[string]string:
		aMap := (*Node)(NewMap())
		for k, v := range actual {
			aMap.Put(k, v)
		}
		return (*yaml.Node)(aMap)
	case []interface{}:
		aSlice := (*Node)(NewSlice())
		for j := range actual {
			aSlice.Append(actual[j])
		}
		return (*yaml.Node)(aSlice)
	case []string:
		aSlice := (*Node)(NewSlice())
		for j := range actual {
			aSlice.Append(actual[j])
		}
		return (*yaml.Node)(aSlice)
	default:
		panic(fmt.Sprintf("not supported yaml.node put type %T", actual))
	}
}

func NewSlice() *yaml.Node {
	return &yaml.Node{
		Kind: yaml.SequenceNode,
		Tag:  "!!seq",
	}
}

func NewMap() *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}
}

func NewDocument() *yaml.Node {
	return &yaml.Node{
		Kind: yaml.DocumentNode,
	}
}

func newScalar(value interface{}) *yaml.Node {

	rType := reflect.TypeOf(value)
	if rType != nil && rType.Kind() == reflect.Ptr {
		rValue := reflect.ValueOf(value)
		if rValue.IsNil() {
			value = nil
		} else {
			value = reflect.ValueOf(value).Elem().Interface()
		}
	}
	if value == nil {
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!null",
			Value: "",
		}

	}
	tag := ""

	switch value.(type) {
	case string, []byte:
		tag = "!!str"
	case int, int64, uint64:
		tag = "!!int"
	case float64, float32:
		tag = "!!float"
	case bool:
		tag = "!!bool"
	default:
		tag = "!!str"
	}
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   tag,
		Value: toolbox.AsString(value),
	}
}
