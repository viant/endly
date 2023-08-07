package xunit

// Testsuite represents an error test-suite nodes
type Testsuite struct {
	Name string `xml:"name,attr,omitempty" yaml:"name,omitempty"  json:"name,omitempty" `

	Errors       string `xml:"errors,attr,omitempty" yaml:"errors,omitempty"  json:"errors,omitempty" `
	ErrorsDetail string `xml:"errors-detail,attr,omitempty" yaml:"errors-detail,omitempty"  json:"errors-detail,omitempty" `

	Failures       string `xml:"failures,attr,omitempty" yaml:"failures,omitempty"  json:"failures,omitempty" `
	FailuresDetail string `xml:"failures-detail,attr,omitempty" yaml:"failures-detail,omitempty"  json:"failures-detail,omitempty" `

	Tests     string `xml:"tests,attr" yaml:"tests,omitempty"  json:"tests,omitempty" `
	TestCases string `xml:"test-cases,attr,omitempty" yaml:"test-cases,omitempty"  json:"test-cases,omitempty" `
	Reports   string `xml:"reports,attr" yaml:"reports,omitempty"  json:"reports,omitempty" `

	Time     string      `xml:"time,attr,omitempty" yaml:"time,omitempty"  json:"time,omitempty" `
	TestCase []*TestCase `xml:"testcase" yaml:"test-case,omitempty"  json:"test-case,omitempty" `
}

func NewTestsuite() *Testsuite {
	return &Testsuite{
		TestCase: make([]*TestCase, 0),
	}
}
