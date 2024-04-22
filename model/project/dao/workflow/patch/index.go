package patch

type WorkflowSlice []*Workflow
type IndexedWorkflow map[string]*Workflow

func (c WorkflowSlice) IndexById() IndexedWorkflow {
	var result = IndexedWorkflow{}
	for i, item := range c {
		if item != nil {
			result[item.Id] = c[i]
		}
	}
	return result
}

type TaskSlice []*Task
type IndexedTask map[string]*Task

func (c TaskSlice) IndexById() IndexedTask {
	var result = IndexedTask{}
	for i, item := range c {
		if item != nil {
			result[item.Id] = c[i]
		}
	}
	return result
}

type AssetSlice []*Asset
type IndexedAsset map[string]*Asset

func (c AssetSlice) IndexById() IndexedAsset {
	var result = IndexedAsset{}
	for i, item := range c {
		if item != nil {
			result[item.Id] = c[i]
		}
	}
	return result
}

type ProjectSlice []*Project
type IndexedProject map[string]*Project

func (c ProjectSlice) IndexById() IndexedProject {
	var result = IndexedProject{}
	for i, item := range c {
		if item != nil {
			result[item.Id] = c[i]
		}
	}
	return result
}

func (c IndexedWorkflow) Has(key string) bool {
	_, ok := c[key]
	return ok
}
func (c IndexedTask) Has(key string) bool {
	_, ok := c[key]
	return ok
}
func (c IndexedAsset) Has(key string) bool {
	_, ok := c[key]
	return ok
}
func (c IndexedProject) Has(key string) bool {
	_, ok := c[key]
	return ok
}
