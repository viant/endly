package xunit

//Nodes represents an error xunit nodes
type Nodes struct {
	Expected string   `xml:"expected,attr,omitempty"  yaml:"expected,omitempty"  json:"expected,omitempty" `
	Result   string   `xml:"result,attr,omitempty"  yaml:"result,omitempty"  json:"result,omitempty"  `
	Nodes    []*Nodes `xml:"nodes,omitempty"  yaml:"nodes,omitempty"  json:"nodes,omitempty" `
	Error    *Error   `xml:"error,omitempty"  yaml:"error,omitempty"  json:"error,omitempty" `
}

//NewNodes creates a new nodes
func NewNodes() *Nodes {
	return &Nodes{
		Nodes: []*Nodes{},
	}
}
