package model

import (
	"strings"
	"path"
)

//WorkflowSelector represents an expression to invoke workflow with all or specified task:  URL[:tasks]
type WorkflowSelector string

//URL returns workflow url
func (s WorkflowSelector) URL() string {
	URL, _, _ := s.split()
	return URL
}

//IsRelative returns true if selector is relative path
func (s WorkflowSelector) IsRelative() bool {
	URL := s.URL()
	if strings.Contains(URL, "://") || strings.HasPrefix(URL, "/") {
		return false
	}
	return true
}



//split returns selector URL, name and tasks
func (s WorkflowSelector) split() (URL, name, tasks string) {
	var sel = string(s)
	protoPosition := strings.LastIndex(sel, "://")
	taskPosition := strings.LastIndex(sel, ":")
	if protoPosition != -1 {
		taskPosition = -1
		selWithoutProto := string(sel[protoPosition+3:])
		if position := strings.LastIndex(selWithoutProto, ":"); position != -1 {
			taskPosition = protoPosition + 3 + position
		}
	}
	URL = sel
	tasks = "*"
	if taskPosition != -1 {
		tasks = string(URL[taskPosition+1:])
		URL = string(URL[:taskPosition])

	}
	var ext = path.Ext(URL)
	if ext == "" {
		_, name = path.Split(URL)
		URL += ".csv"
	} else {
		_, name = path.Split(string(URL[:len(URL)-len(ext)]))
	}
	return URL, name, tasks
}

//Name returns selector workflow name
func (s WorkflowSelector) Name() string {
	_, name, _ := s.split()
	return name
}

//TasksSelector returns selector tasks
func (s WorkflowSelector) Tasks() string {
	_, _, tasks := s.split()
	return tasks

}


//ActionSelector represents an expression to invoke endly action:  service.Action (for workflow service workflow keyword can be skipped)
type ActionSelector string

//Action returns action
func (s ActionSelector) Action() string {
	pair := strings.Split(string(s), ".")
	if len(pair) == 2 {
		return pair[1]
	}
	return string(s)
}

//Service returns service
func (s ActionSelector) Service() string {
	pair := strings.Split(string(s), ".")
	if len(pair) == 2 {
		return pair[0]
	}
	return string("workflow")
}

//TasksSelector represents a task selector
type TasksSelector string

//Tasks return tasks
func (t *TasksSelector) Tasks() []string {
	var result = strings.Split(string(*t), ",")
	for i, item := range result {
		result[i] = strings.TrimSpace(item)
	}
	return result
}

//RunAll returns true if no individual tasks are selected
func (t *TasksSelector) RunAll() bool {
	return *t == "" || *t == "*"
}
