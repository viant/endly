package yml

import "gopkg.in/yaml.v3"

type Nodes []*yaml.Node

func (n Nodes) Map() (*yaml.Node, error) {
	result := NewMap()
	for _, item := range n {
		result.Content = append(result.Content, item.Content...)
	}
	return result, nil
}

func (n Nodes) LookupNode(name string) *yaml.Node {
	if len(n) == 0 {
		return nil
	}
	for i := 0; i < len(n); i++ {
		return n[i]

	}
	return nil
}

func (n Nodes) LookupValueNode(name string) *yaml.Node {
	if len(n) == 0 {
		return nil
	}
	for i := 0; i < len(n); i++ {
		node := n[i]
		if name == node.Value {
			return n[i+1]
		}
	}
	return nil
}

func (n Nodes) LookupNodeValue(name string) string {
	node := n.LookupValueNode(name)
	if node != nil {
		return node.Value
	}
	return ""
}

func (n Nodes) FilterNode(name string) []*yaml.Node {
	var result = make([]*yaml.Node, 0)
	for i := 0; i < len(n); {
		node := n[i]
		i++
		if name == node.Value {
			i++
			continue
		}
		result = append(result, node)
	}
	return result
}
func (n Nodes) Filter(names map[string]bool) []*yaml.Node {
	var result = make([]*yaml.Node, 0)
	for i := 0; i < len(n); {
		node := n[i]
		i++
		if names[node.Value] {
			i++
			continue
		}
		result = append(result, node)
	}
	return result
}

func (n Nodes) AppendScalar(key string) []*yaml.Node {
	return append(n, &yaml.Node{Value: key, Kind: yaml.ScalarNode, Tag: "!!str"})
}
func (n Nodes) Append(key, values string) []*yaml.Node {
	return append(n, &yaml.Node{Value: key, Kind: yaml.ScalarNode, Tag: "!!str"},
		&yaml.Node{Value: values, Kind: yaml.ScalarNode, Tag: "!!str"})
}
