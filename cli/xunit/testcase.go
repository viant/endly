package xunit

// TestCase represents an error test-case nodes
type TestCase struct {
	Name string `xml:"name,attr,omitempty" yaml:"name,omitempty"  json:"name,omitempty"`

	Label string `xml:"label,attr,omitempty" yaml:"label,omitempty"  json:"label,omitempty"`
	Skip  string `xml:"skip,attr,omitempty"  yaml:"skip,omitempty"  json:"skip,omitempty"`

	Tests string `xml:"tests,attr,omitempty"  yaml:"tests,omitempty"  json:"tests,omitempty"`

	Failures       string `xml:"failures,attr,omitempty" yaml:"failures,omitempty"  json:"failures,omitempty" `
	FailuresDetail string `xml:"failures-detail,attr,omitempty"  yaml:"failures-detail,omitempty"  json:"failures-detail,omitempty"`
	Errors         string `xml:"errors,attr,omitempty"  yaml:"errors,omitempty"  json:"errors,omitempty"`
	ErrorsDetail   string `xml:"errors-detail,attr,omitempty"  yaml:"errors-detail,omitempty"  json:"errors-detail,omitempty"`
	TestCases      string `xml:"test-cases,attr,omitempty"  yaml:"test-cases,omitempty"  json:"test-cases,omitempty"`
	Reports        string `xml:"reports,attr,omitempty"  yaml:"reports,omitempty"  json:"reports,omitempty"`
	Time           string `xml:"time,attr,omitempty"  yaml:"time,omitempty"  json:"time,omitempty"`
	Nodes          *Nodes `xml:"nodes,omitempty"  yaml:"nodes,omitempty"  json:"nodes,omitempty"`
	Sysout         string `xml:"sysout,omitempty"  yaml:"sysout,omitempty"  json:"sysout,omitempty"`
	Syserr         string `xml:"syserr,omitempty"  yaml:"syserr,omitempty"  json:"syserr,omitempty"`
}

// NewTestCase creates a new test case
func NewTestCase() *TestCase {
	return &TestCase{}
}
