package project

// Bundle represents a project bundle
type Bundle struct {
	Project *Project
	*Workflow
	tasks        []*Task
	assets       []*Asset
	SubWorkflows []*Bundle
	Templates    map[string][]*Bundle
}

// Workflows returns all workflows
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

// Assets returns all assets
func (b *Bundle) Assets() []*Asset {
	var result []*Asset
	result = append(result, b.assets...)
	for _, sub := range b.SubWorkflows {
		result = append(result, sub.Assets()...)
	}
	for _, tmpls := range b.Templates {
		for _, tmpl := range tmpls {
			result = append(result, tmpl.Assets()...)
		}
	}
	return result
}

// Projects returns all projects
func (b *Bundle) Projects() []*Project {
	var result []*Project
	result = append(result, b.Project)
	return result
}

// Tasks returns all tasks
func (b *Bundle) Tasks() []*Task {
	var result []*Task
	result = append(result, b.tasks...)
	for _, sub := range b.SubWorkflows {
		result = append(result, sub.Tasks()...)
	}
	for _, tmpls := range b.Templates {
		for _, tmpl := range tmpls {
			result = append(result, tmpl.Tasks()...)
		}
	}
	return result
}

func (b *Bundle) LookupWorkflow(id string) *Workflow {
	if b.Workflow.ID == id {
		return b.Workflow
	}
	for _, sub := range b.SubWorkflows {
		if w := sub.LookupWorkflow(id); w != nil {
			return w
		}
	}
	for _, tmpls := range b.Templates {
		for _, tmpl := range tmpls {
			if w := tmpl.LookupWorkflow(id); w != nil {
				return w
			}
		}
	}
	return nil
}

func (b *Bundle) AppendAsset(asset *Asset) {
	b.assets = append(b.assets, asset)
	workflow := b.LookupWorkflow(asset.WorkflowID)
	if workflow == nil {
		workflow = b.Workflow
	}
	workflow.Assets = append(workflow.Assets, asset)
}

func (b *Bundle) AppendTask(task *Task) {
	b.tasks = append(b.tasks, task)
	b.Workflow.Steps = append(b.Workflow.Steps, task)
}
