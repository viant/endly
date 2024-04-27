package node

import (
	"fmt"
	"golang.org/x/net/html"
	"strings"
)

type Node struct {
	*html.Node
	selectors                        map[string]string
	Parent, PrevSibling, NextSibling *Node
}

func (n *Node) IsBaseNode() bool {
	switch strings.ToLower(n.Data) {
	case "input", "textarea", "button", "select", "a", "label":
		return true
	}
	return false
}

func (n *Node) Selectors(attributes []string, exclude ...string) []string {
	var result []string
	for _, k := range attributes {
		v, ok := n.selectors[k]
		if !ok {
			continue
		}
		if isExcluded(v, exclude) {
			continue
		}
		result = append(result, v)
	}

	if !n.IsBaseNode() {
		if n.PrevSibling != nil {
			if n.PrevSibling.IsBaseNode() {
				for _, sel := range n.PrevSibling.selectors {
					result = append(result, fmt.Sprintf("%s[preceding-sibling::%s]", n.Data, sel))
				}
			}
		}
		tableNode := getTableNode(n)
		rowNode := getRowNode(n)
		result = n.buildTableBaseSelector(attributes, exclude, tableNode, rowNode, result)

		result = n.buildAncestorBasedSelector(attributes, exclude, result)
	}

	return result
}

func (n *Node) buildAncestorBasedSelector(attributes []string, exclude []string, result []string) []string {
	if parent := n.Parent; parent != nil {
		parentSelector := parent.Selectors(attributes, exclude...)
		if len(parentSelector) == 0 {
			if grandParent := parent.Parent; grandParent != nil {
				if grandParentSelector := grandParent.Selectors(attributes, exclude...); len(grandParentSelector) > 0 {
					var tempSelectors []string
					if len(result) > 0 {
						for _, leaf := range result {
							tempSelectors = append(tempSelectors, grandParentSelector[0]+"/"+parent.Data+"/"+leaf[2:])
						}
						result = append(result, tempSelectors...)
					}
				}
			}
		} else {
			var tempSelectors []string
			for _, selector := range parentSelector {
				for _, leaf := range result {
					tempSelectors = append(tempSelectors, selector+"/"+leaf[2:])
				}
			}
			result = append(result, tempSelectors...)
		}
	}

	if len(result) == 0 {
		switch n.Data {
		case "path", "svg", "img":
			parent := n.Parent

			if grandParent := parent.Parent; grandParent != nil {
				grandParentSelector := grandParent.Selectors(attributes, exclude...)
				if len(grandParentSelector) > 0 {
					result = append(result, grandParentSelector[0]+"/"+parent.Data+"/"+n.Data)
				}
			} else {
				parentSelector := grandParent.Selectors(attributes, exclude...)
				if len(parentSelector) > 0 {
					result = append(result, parentSelector[0]+"/"+n.Data)
				}
			}
		}
	}

	return result
}

func (n *Node) buildTableBaseSelector(attributes []string, exclude []string, tableNode *Node, rowNode *Node, result []string) []string {
	if tableNode != nil && rowNode != nil {
		cellNode := getCellData(n)
		var tempSelectors []string
		for _, leaf := range result {
			for _, selector := range tableNode.Selectors(attributes, exclude...) {
				rowIndex := getRowIndex(tableNode.Node, rowNode.Node)
				cellSelector := selector + fmt.Sprintf("//tr[%v]", rowIndex)
				if cellNode != nil {
					cellIndex := getCellIndex(rowNode.Node, cellNode.Node)
					cellSelector += fmt.Sprintf("//%v[%v]", cellNode.Data, cellIndex)
				}
				if n.Data == "td" || n.Data == "th" {
					tempSelectors = append(tempSelectors, cellSelector)
					break
				} else {
					cellSelector += leaf
					tempSelectors = append(tempSelectors, cellSelector)
				}
			}
		}
		result = append(result, tempSelectors...)

	}
	return result
}

func getTableNode(n *Node) *Node {
	if n.Parent != nil {
		if n.Parent.Data == "table" {
			return n.Parent
		}
		return getTableNode(n.Parent)
	}
	return nil
}

func getRowNode(n *Node) *Node {
	if n.Data == "tr" {
		return n
	}
	if n.Parent != nil {
		if n.Parent.Data == "tr" {
			return n.Parent
		}
		return getRowNode(n.Parent)
	}
	return nil
}

// getRowIndex finds the zero-based index of row within its specific section (thead, tbody, tfoot) in the table.
// It returns -1 if the row is not found within its parent section.
func getRowIndex(table, row *html.Node) int {

	// Iterate through children of the identified section
	index := 0
	for child := table.FirstChild; child != nil; child = child.NextSibling {
		switch child.Data {
		case "thead", "tbody", "tfoot":
			for rowCandidate := child.FirstChild; rowCandidate != nil; rowCandidate = rowCandidate.NextSibling {
				if rowCandidate.Type == html.ElementNode && rowCandidate.Data == "tr" {
					if isNodeEqual(row, rowCandidate) {
						return index
					}
					index++
				}
			}
		case "tr":
			if isNodeEqual(row, child) {
				return index
			}
			index++
		}
	}
	return -1 // Return -1 if the row is not found
}

func getCellData(n *Node) *Node {
	if n.Data == "td" || n.Data == "th" {
		return n
	}
	if n.Parent != nil {
		switch n.Parent.Data {
		case "td", "th":
			return n.Parent
		}
		return getCellData(n.Parent)
	}
	return nil
}

// getCellIndex finds the zero-based index of a cell (td or th) within its parent row (tr).
// It returns -1 if the cell is not found within the row.
func getCellIndex(row, cell *html.Node) int {
	index := 0
	for sibling := row.FirstChild; sibling != nil; sibling = sibling.NextSibling {
		// Check if the sibling is a cell element (either td or th)
		if sibling.Type == html.ElementNode && (sibling.Data == "td" || sibling.Data == "th") {
			if isNodeEqual(sibling, cell) {
				return index
			}
			index++
		}
	}
	return -1 // Return -1 if the cell is not found
}

func isExcluded(k string, exclude []string) bool {
	if len(exclude) == 0 {
		return false
	}
	for _, candidate := range exclude {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if strings.Contains(k, candidate) {
			return true
		}
	}
	return false
}
