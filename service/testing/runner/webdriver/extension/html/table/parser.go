package table

import (
	"bytes"
	"golang.org/x/net/html"
	"strings"
)

type Parser struct{ Document *html.Node }
type parsedRow struct {
	cells  []string
	header bool
}

func NewParser(content string) (*Parser, error) {
	if strings.HasPrefix(strings.TrimSpace(content), "<table") {
		content = "<html><body>" + content + "</body></html>"
	}
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, err
	}
	return &Parser{Document: doc}, nil
}
func (p *Parser) rows() []parsedRow {
	var root *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if root != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "table" {
			root = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(p.Document)
	if root == nil {
		return nil
	}
	var result []parsedRow
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, inHead bool) {
		if n != root && n.Type == html.ElementNode && n.Data == "table" {
			return
		}
		inHead = inHead || n.Type == html.ElementNode && n.Data == "thead"
		if n.Type == html.ElementNode && n.Data == "tr" {
			row := parsedRow{header: true}
			for cell := n.FirstChild; cell != nil; cell = cell.NextSibling {
				if cell.Type != html.ElementNode || (cell.Data != "td" && cell.Data != "th") {
					continue
				}
				row.cells = append(row.cells, p.extractText(cell))
				if cell.Data == "td" && !inHead {
					row.header = false
				}
			}
			if len(row.cells) > 0 {
				result = append(result, row)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inHead)
		}
	}
	walk(root, false)
	return result
}

// ParseTable returns rendered cell text, including header rows, from one table.
func (p *Parser) ParseTable() [][]string {
	var result [][]string
	for _, row := range p.rows() {
		result = append(result, row.cells)
	}
	return result
}
func (p *Parser) extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	if n.Type == html.ElementNode && (n.Data == "table" || n.Data == "script" || n.Data == "style") {
		return ""
	}
	if n.Type == html.ElementNode && n.Data == "br" {
		return "\n"
	}
	var b bytes.Buffer
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(p.extractText(c))
	}
	return strings.TrimSpace(b.String())
}
