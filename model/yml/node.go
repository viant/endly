package yml

import "gopkg.in/yaml.v3"

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
