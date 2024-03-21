package transfer

type Bundle struct {
	URI     string
	Project *Project
	*Workflow
	Tasks        []*Task
	Assets       []*Asset
	SubWorkflows []*Bundle
	Templates    map[string][]*Bundle
}
