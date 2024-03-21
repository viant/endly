package model

// Task represents a group of action
type Task struct {
	*AbstractNode `yaml:",inline"` //abstract node
	Actions       []*Action        ` yaml:",omitempty"` //actions
	*TasksNode    ` yaml:",inline"`
	Fail          bool      ` yaml:",omitempty"` //controls if return fail status workflow on catch task
	Template      *Template ` yaml:",omitempty"`
	//internal only for inline workflow meta data

	multiAction bool //flag directing grouping actions (otherwise each action has its own task)
	//publish data in parent workflow
	data map[string]string

	//these attribute if present dynamically load actions from subpath
	tagRange string
	subpath  string
}

//func (t Task) MarshalYAML() (interface{}, error) {
//	var result = make(map[string]interface{})
//
//
//	return result, nil
//}

func (t *Task) Clone() *Task {
	var result = *t
	result.Actions = make([]*Action, len(t.Actions))
	copy(result.Actions, t.Actions)
	for i, item := range result.Actions {
		result.Actions[i] = item.Clone()
	}
	result.TasksNode = t.TasksNode.Clone()
	result.AbstractNode = t.AbstractNode.Clone()
	return &result
}

func (t *Task) init() error {
	if len(t.Actions) == 0 {
		t.Actions = []*Action{}
	}
	for _, action := range t.Actions {
		if t.Logging != nil && action.Logging == nil {
			action.Logging = t.Logging
		}

		if err := action.Init(); err != nil {
			return err
		}
	}
	if t.TasksNode == nil {
		return nil
	}
	if len(t.Tasks) > 0 {
		for _, task := range t.Tasks {
			if t.Logging != nil && task.Logging == nil {
				task.Logging = t.Logging
			}
			if err := task.init(); err != nil {
				return err
			}
		}
	}
	return nil
}

// HasTagID checks if task has supplied tagIDs
func (t *Task) HasTagID(tagIDs map[string]bool) bool {
	if tagIDs == nil {
		return false
	}
	for _, action := range t.Actions {
		if tagIDs[action.TagID] {
			return true
		}
	}
	return false
}

// AsyncActions returns async actions
func (t *Task) AsyncActions() []*Action {
	var result = make([]*Action, 0)
	for _, candidate := range t.Actions {

		if candidate.Async {
			if candidate.Repeat > 1 {
				repeat := candidate.Repeat
				action := candidate.Clone()
				_ = action.Init()
				action.Repeat = 1
				for i := 0; i < repeat; i++ {
					result = append(result, action)
				}
			} else {
				result = append(result, candidate)
			}
		}
	}
	return result
}

// NewTask creates a new task
func NewTask(name string, multiAction bool) *Task {
	return &Task{
		AbstractNode: &AbstractNode{
			Name: name,
		},
		Actions:     make([]*Action, 0),
		multiAction: multiAction,
		TasksNode: &TasksNode{
			Tasks: make([]*Task, 0),
		},
	}
}
