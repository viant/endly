package xml

import (
	"encoding/xml"
	"io"
	"strings"
)

type (
	Map      map[string]interface{}
	MapEntry struct {
		XMLName  xml.Name
		Value    string     `xml:",chardata"`
		Attrs    []xml.Attr `xml:",any,attr"`
		Children []MapEntry `xml:",any"`
	}

	Node struct {
		Name     string            `json:",omitempty"`
		Attrs    map[string]string `json:",omitempty"`
		Value    string            `json:",omitempty"`
		Children []*Node           `json:",omitempty"`
	}
)

func (m MapEntry) AsNode() *Node {
	name := m.XMLName.Local
	if m.XMLName.Space != "" {
		name = m.XMLName.Space + ":" + name
	}
	result := &Node{Name: name}
	if len(m.Attrs) > 0 {
		result.Attrs = make(map[string]string)
	}
	for _, attr := range m.Attrs {
		key := attr.Name.Local
		if attr.Name.Space != "" {
			key = attr.Name.Space + ":" + key
		}
		result.Attrs[key] = strings.TrimSpace(attr.Value)
	}

	if m.Value != "" {
		result.Value = strings.TrimSpace(m.Value)
	}
	for _, child := range m.Children {
		result.Children = append(result.Children, child.AsNode())
	}

	return result
}

func (n *Node) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*n = Node{Name: start.Name.Local}
	if len(start.Attr) > 0 {
		n.Attrs = make(map[string]string)
	}
	for _, attr := range start.Attr {
		key := attr.Name.Local
		if attr.Name.Space != "" {
			key = attr.Name.Space + ":" + key
		}
		n.Attrs[key] = strings.TrimSpace(attr.Value)
	}
	for {
		var e MapEntry
		err := d.Decode(&e)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		n.Children = append(n.Children, e.AsNode())
	}
	return nil
}
