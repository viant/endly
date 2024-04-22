package table

import (
	"bytes"
	"golang.org/x/net/html"
	"strings"
)

// Parser handles parsing of HTML tables
type Parser struct {
	Document *html.Node
}

// NewParser creates a new parser instance
func NewParser(htmlContent string) (*Parser, error) {
	if strings.HasPrefix(htmlContent, "<table") {
		htmlContent = "<html><body>" + htmlContent + "</body></html>"
	}
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}
	return &Parser{Document: doc}, nil
}

// ParseTable parses the HTML table and returns a 2D array of strings
func (p *Parser) ParseTable() [][]string {
	var tableRows [][]string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			row := p.parseTableRow(n)
			if len(row) > 0 {
				tableRows = append(tableRows, row)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(p.Document)
	return tableRows
}

// parseTableRow extracts cells from a table row
func (p *Parser) parseTableRow(tr *html.Node) []string {
	var row []string
	for cell := tr.FirstChild; cell != nil; cell = cell.NextSibling {
		if cell.Type == html.ElementNode && cell.Data == "td" {
			text := p.extractText(cell)
			row = append(row, text)
		}
	}
	return row
}

// extractText recursively extracts text from a node
func (p *Parser) extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var buf bytes.Buffer
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		buf.WriteString(p.extractText(c))
	}
	return strings.TrimSpace(buf.String())
}
