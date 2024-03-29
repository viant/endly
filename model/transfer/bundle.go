package transfer

type Bundle struct {
	Project *Project
	*Workflow
	Tasks        []*Task
	Assets       []*Asset
	SubWorkflows []*Bundle
	Templates    map[string][]*Bundle
}


func (b *Bundle) Workflows() []*Workflow {
	var result []*Workflow
	result = append(result, b.Workflow)
	for _, sub := range b.SubWorkflows {
		result = append(result, sub.Workflows()...)
	}
	for _, tmpls := range b.Templates {
		for _, tmpl := range tmpls {
			result = append(result, tmpl.Workflows()...)
		}
	}
	return result
}

func (b *Bundle) AllAssets() []*Asset {
	var result []*Asset
	result = append(result, b.Assets...)
	for _, sub := range b.SubWorkflows {
		result = append(result, sub.AllAssets()...)
	}
	for _, tmpls := range b.Templates {
		for _, tmpl := range tmpls {
			result = append(result, tmpl.AllAssets()...)
		}
	}
	return result
}

func (b *Bundle) Projects() []*Project {
	var result []*Project
	result = append(result, b.Project)
	return result
}

func (b *Bundle) AllTasks() []*Task {
	var result []*Task
	result = append(result, b.Tasks...)
	for _, sub := range b.SubWorkflows {
		result = append(result, sub.AllTasks()...)
	}
	for _, tmpls := range b.Templates {
		for _, tmpl := range tmpls {
			result = append(result, tmpl.AllTasks()...)
		}
	}
	return result
}