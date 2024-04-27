package node

import (
	"golang.org/x/net/html"
	"strings"
)

func tableContextNode() *html.Node {
	if nodes, err := html.ParseFragment(strings.NewReader("<table></table>"), nil); err == nil {
		return nodes[0].LastChild.LastChild
	}
	return nil
}

func fieldsetContextNode() *html.Node {
	if nodes, err := html.ParseFragment(strings.NewReader("<fieldset></fieldset>"), nil); err == nil {
		return nodes[0].LastChild.LastChild
	}
	return nil
}

func tableRowContextNode() *html.Node {
	if nodes, err := html.ParseFragment(strings.NewReader("<table><tr></tr></table>"), nil); err == nil {
		return nodes[0].LastChild.LastChild.LastChild.LastChild
	}
	return nil
}

func divContextNode() *html.Node {
	if nodes, err := html.ParseFragment(strings.NewReader("<div></div"), nil); err == nil {
		return nodes[0].LastChild.LastChild
	}
	return nil
}

func getContextNode(fragment string) *html.Node {

	if strings.HasPrefix(fragment, "<td") {
		return tableRowContextNode()
	}
	if strings.HasPrefix(fragment, "<th") {
		return tableRowContextNode()
	}
	if strings.HasPrefix(fragment, "<tr") {
		return tableContextNode()
	}
	if strings.HasPrefix(fragment, "<tbody") {
		return tableContextNode()
	}
	if strings.HasPrefix(fragment, "<thead") {
		return tableContextNode()
	}
	if strings.HasPrefix(fragment, "<legend") {
		return fieldsetContextNode()
	}
	return divContextNode()
}
