package model

import (
	"github.com/viant/endly/model/graph/yml"
	"gopkg.in/yaml.v3"
)

func (t *Workflow) MarshalYAML() (interface{}, error) {
	type workflow Workflow
	orig := (*workflow)(t)
	var node = yaml.Node{}
	if err := node.Encode(&orig); err != nil {
		return nil, err
	}
	node.Content = yml.Nodes(node.Content).FilterNode("source")
	updateWorkflowFormat(nil, &node)
	return node, nil
}

func updateWorkflowFormat(parent, n *yaml.Node) {
	if len(n.Content) == 0 {
		return
	}
	var adjusted = make([]*yaml.Node, 0, len(n.Content))
	for i := 0; i < len(n.Content); {
		node := n.Content[i]
		i++
		updateWorkflowFormat(n, node)
		if parent == nil {
			continue
		}
		isTaskNode := node.Value == "tasks"
		if node.Kind == yaml.ScalarNode && isTaskNode {
			reduced := make([]*yaml.Node, 0)
			childContent := n.Content[i].Content
			for j := 0; j < len(childContent); j++ {
				reduceTaskNodes(childContent, j, n, &reduced)
			}
			adjusted = append(adjusted, reduced...)
			i++
			continue
		}
		adjusted = append(adjusted, node)
	}
	if parent != nil && len(adjusted) > 0 {
		n.Content = adjusted
	}
}

func reduceTaskNodes(childContent []*yaml.Node, j int, n *yaml.Node, reduced *[]*yaml.Node) {
	key := childContent[j].Content[1]
	content := childContent[j].Content[2:]
	holder := &yaml.Node{
		Tag:     "!!map",
		Kind:    yaml.MappingNode,
		Content: content,
	}

	var requestNodes = make([]*yaml.Node, 0)
	actionsNodes := yml.Nodes(content).LookupValueNode("actions")
	if actionsNodes != nil {
		actionNodes := actionsNodes.Content[0].Content
		action := yml.Nodes(actionNodes).LookupNodeValue("action")
		if serviceValue := yml.Nodes(actionNodes).LookupNodeValue("service"); serviceValue != "" && serviceValue != "workflow" {
			actionNode := yml.Nodes(actionNodes).LookupValueNode("action")
			actionNode.Value = serviceValue + ":" + actionNode.Value
			action = actionNode.Value
		}
		var requestURL string
		var tasks string
		if requestNode := yml.Nodes(actionNodes).LookupValueNode("request"); requestNode != nil {
			requestURL = yml.Nodes(requestNode.Content).LookupNodeValue("AssetURL")
			tasks = yml.Nodes(requestNode.Content).LookupNodeValue("tasks")
		}
		filter := actionInternalNodes
		if action == "run" || action == "workflow:run" {
			filter = actionInternalNodesWithRequest
			requestNodes = yml.Nodes(requestNodes).Append("request", "@"+requestURL)
			if tasks != "" {
				requestNodes = yml.Nodes(requestNodes).Append("tasks", tasks)
			}
		}
		holder.Content = yml.Nodes(actionNodes).Filter(filter)
	}

	updateWorkflowFormat(n, holder)
	if len(requestNodes) > 0 {
		holder.Content = append(holder.Content, requestNodes...)
	}
	*reduced = append(*reduced, key)
	*reduced = append(*reduced, holder)
}

var actionInternalNodes = map[string]bool{
	"name":    true,
	"tagid":   true,
	"repeat":  true,
	"tag":     true,
	"service": true,
}

var actionInternalNodesWithRequest = map[string]bool{
	"name":    true,
	"tagid":   true,
	"repeat":  true,
	"tag":     true,
	"service": true,
	"request": true,
}
