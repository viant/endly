package endly

type Asset struct {
	Name           string
	URL            string
	Body           string
	CredentialFile string
}

type Variable struct {
	Name     string
	Value    interface{}
	DataType string
}

type UseCase struct {
	Name        string
	URL         string
	Feature     string
	Requirement string
	Steps       []TestStep
}

type TestStep map[string]interface{}

type Datastore struct {
}

type ApplicationService struct {
	URL                string
	HttpMethod         string
	RequestTemplate    interface{}
	ResponseToStateMap map[string]string
}

type Deployment struct {
}

type TestPlan struct {
	Name       string
	Services   []*Service
	Deployment *Deployment
	Asset      []*Asset
	Variables  []*Variable
	UseCases   []*UseCase
}

type TestPlanRegistry struct {
}
