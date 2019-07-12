package model

import (
	"fmt"
	"strings"
)

//TasksNode represents a task node
type TasksNode struct {
	Tasks        []*Task //sub tasks
	OnErrorTask  string  //task that will run if error occur, the final workflow will return this task response
	Fail         bool    //controls if return fail status workflow on catch task
	DeferredTask string  //task that will always run if there has been previous  error or not
}

//Select selects tasks matching supplied selector
func (t *TasksNode) Select(selector TasksSelector) *TasksNode {
	if selector.RunAll() {
		return t
	}
	var allowed = make(map[string]bool)
	for _, task := range selector.Tasks() {
		allowed[task] = true
	}
	var result = &TasksNode{
		OnErrorTask:  t.OnErrorTask,
		DeferredTask: t.DeferredTask,
		Tasks:        []*Task{},
	}

	if result.DeferredTask != "" {
		allowed[result.DeferredTask] = true
	}
	if result.OnErrorTask != "" {
		allowed[result.OnErrorTask] = true
	}

	for _, task := range t.Tasks {

		if task.TasksNode != nil && len(task.Tasks) > 0 {
			if allowed[task.Name] {
				result.Tasks = append(result.Tasks, task.Tasks...)
			} else {
				var selected = task.TasksNode.Select(selector)
				if len(selected.Tasks) > 0 {
					result.Tasks = append(result.Tasks, selected.Tasks...)
				}
			}
		} else if allowed[task.Name] {
			result.Tasks = append(result.Tasks, task)
		}
	}
	return result
}

//Task returns a task for supplied name
func (t *TasksNode) Task(name string) (*Task, error) {
	if len(t.Tasks) == 0 {
		return nil, fmt.Errorf("failed to lookup task: %v", name)
	}
	name = strings.TrimSpace(name)
	for _, candidate := range t.Tasks {
		if candidate.Name == name {
			return candidate, nil
		}
		if candidate.TasksNode != nil {
			if result, err := candidate.Task(name); err == nil {
				return result, nil
			}
		}
	}
	return nil, fmt.Errorf("failed to lookup task: %v", name)
}

//Task returns a task for supplied name
func (t *TasksNode) Has(name string) bool {
	if len(t.Tasks) == 0 {
		return false
	}
	_, err := t.Task(name)
	return err == nil
}
