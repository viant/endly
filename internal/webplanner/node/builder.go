package node

import (
	"fmt"
	"golang.org/x/net/html"
	"strings"
)

type Builder struct {
	attributes []string
}

func NewBuilder(attributes ...string) *Builder {
	ret := &Builder{}
	for _, attr := range attributes {
		attribute := strings.TrimSpace(attr)
		if attribute == "" {
			continue
		}
		ret.attributes = append(ret.attributes, attribute)
	}

	return ret
}

func (b *Builder) Build(holderHTML, targetHTML string) (*Node, error) {
	if holderHTML == "" {
		holderHTML = targetHTML
	}
	holder, err := b.parseHTMLFragment(holderHTML)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML content: %v", err)
	}
	target, err := b.parseHTMLFragment(targetHTML)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML content: %v", err)
	}

	holder.Parent = nil
	root := &Node{Node: holder}

	b.buildXPathForNode(root)
	targetNode := traverseNode(root, func(n *Node) {
		b.buildXPathForNode(n)
	}, target)
	return targetNode, nil
}

func (b *Builder) parseHTMLFragment(HTMLFragment string) (*html.Node, error) {
	targetDoc, err := html.ParseFragment(strings.NewReader(HTMLFragment), getContextNode(HTMLFragment))
	if err != nil {
		return nil, err
	}
	if len(targetDoc) == 0 {
		if len(HTMLFragment) > 1000 {
			HTMLFragment = HTMLFragment[:1000]
		}
		fmt.Printf("failed to parse HTML fragment: %s\n", HTMLFragment)
		return nil, fmt.Errorf("failed to parse HTML fragment: %s", HTMLFragment)

	}
	return targetDoc[0], nil
}

func (b *Builder) buildXPathForNode(n *Node) {

	if n.Type == html.ElementNode {
		if len(n.selectors) == 0 {
			n.selectors = make(map[string]string)
		}
		b.buildBaseSelector(n)
	}
}

var standardAttributes = []string{"id", "aria-label", "aria-labelledby", "text()", "class"}

func (b *Builder) Attributes() []string {
	if len(b.attributes) == 0 {
		return standardAttributes
	}
	var indexedAttributes = make(map[string]bool)
	var result []string
	for _, attr := range b.attributes {
		indexedAttributes[attr] = true
		if strings.HasPrefix(attr, "-") {
			continue
		}
		result = append(result, attr)
	}
	for _, attr := range standardAttributes {
		if _, ok := indexedAttributes[attr]; ok {
			continue
		}
		if _, ok := indexedAttributes["-"+attr]; ok {
			continue
		}
		result = append(result, attr)
	}

	return result
}

func (b *Builder) buildBaseSelector(n *Node) {
	for _, attr := range b.Attributes() {
		switch attr {
		case "class":
			n.selectors[attr] = fmt.Sprintf("//%s[contains(@class, \"%s\")]", n.Data, attr)

		case "text()":
			if first := n.Node.FirstChild; first != nil {
				if first.Type == html.TextNode && first.Data != "" {
					n.selectors[attr] = fmt.Sprintf("//%s[%s=\"%s\"]", n.Data, attr, first.Data)
				}
			}
		default:
			ariaLabel := getAttribute(n.Node, attr)
			if ariaLabel != "" {
				n.selectors[attr] = fmt.Sprintf("//%s[@%v=\"%s\"]", n.Data, attr, ariaLabel)
			}
		}
	}
}

func isNodeEqual(n1, n2 *html.Node) bool {
	if n1 == nil && n2 == nil {
		return true
	}
	if n1 == nil || n2 == nil {
		return false
	}
	if n1.Type != n2.Type {
		return false
	}
	if n1.Data != n2.Data {
		return false
	}
	if len(n1.Attr) != len(n2.Attr) {
		return false
	}
	for i, a := range n1.Attr {
		if a.Key != n2.Attr[i].Key || a.Val != n2.Attr[i].Val {
			return false
		}
	}
	if n1.FirstChild != nil && n2.FirstChild != nil {
		return isNodeEqual(n1.FirstChild, n2.FirstChild)
	}
	return true
}

func traverseNode(node *Node, h func(n *Node), target *html.Node) *Node {
	var ret *Node
	var prevSibling *Node
	if node.Type == html.ElementNode {
		for c := node.Node.FirstChild; c != nil; c = c.NextSibling {
			htmlNode := *c
			child := &Node{Node: &htmlNode}
			child.Parent = node
			child.PrevSibling = prevSibling
			if prevSibling != nil {
				prevSibling.NextSibling = child
			}
			prevSibling = child

			if isNodeEqual(c, target) {
				ret = child
			}
			h(child)
			if t := traverseNode(child, h, target); t != nil {
				ret = t
			}
		}
	}
	return ret
}

func getAttribute(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
