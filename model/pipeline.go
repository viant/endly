package model

//represents pipelines
type Pipelines []*Pipeline

//Pipeline represents sequential workflow/action execution.
type Pipeline struct {
	Name      string                 `description:"pipeline task name"`
	Workflow  string                 `description:"workflow (URL[:tasks]) selector "`
	Action    string                 `description:"action (service.action) selector "`
	Params    map[string]interface{} `description:"workflow or action parameters"`
	When      string                 `description:"run criteria"`
	Pipelines Pipelines              `description:"workflow or action pipelines"`
}

//Select selects pipelines matching supplied selector
func (p *Pipelines) Select(selector TasksSelector) Pipelines {
	if selector.RunAll() {
		return *p
	}
	var result = make([]*Pipeline, 0)
	for _, task := range selector.Tasks() {
		for _, pipeline := range *p {
			if task == pipeline.Name {
				result = append(result, pipeline)
				continue
			}
			if len(pipeline.Pipelines) > 0 {
				selected := pipeline.Pipelines.Select(selector)
				result = append(result, selected...)
			}
		}
	}
	return result
}
