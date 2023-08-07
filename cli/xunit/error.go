package xunit

// Error represents an error xunit node
type Error struct {
	Type  string `xml:"type,attr,omitempty"  yaml:"type"  json:"type" `
	Value string `xml:"cdata,omitempty"  yaml:"value"  json:"value"`
}
