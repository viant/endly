package model

import (
	"fmt"
	"strings"
)

// TasksNode represents a task node
type TasksNode struct {
	Tasks        Tasks  ` yaml:",omitempty"` //sub tasks
	OnErrorTask  string ` yaml:",omitempty"` //task that will run if error occur, the final workflow will return this task response
	DeferredTask string ` yaml:",omitempty"` //task that will always run if there has been previous  error or not
}

type Tasks []*Task

// Select selects tasks matching supplied selector
func (t *TasksNode) Select(selector TasksSelector) *TasksNode {
	if selector.RunAll() {
		return t
	}
	var result = &TasksNode{
		OnErrorTask:  t.OnErrorTask,
		DeferredTask: t.DeferredTask,
		Tasks:        []*Task{},
	}
	for _, name := range selector.Tasks() {
		task, err := t.Task(name)
		if err != nil {
			continue
		}
		if task.TasksNode != nil && len(task.Tasks) > 0 {
			result.Tasks = append(result.Tasks, task.Tasks...)
		} else {
			result.Tasks = append(result.Tasks, task)
		}
	}
	result.appendControlTask(t, result.OnErrorTask)
	if result.DeferredTask != result.OnErrorTask {
		result.appendControlTask(t, result.DeferredTask)
	}
	return result
}

func (t *TasksNode) appendControlTask(source *TasksNode, name string) {
	if name == "" {
		return
	}
	for _, candidate := range t.Tasks {
		if candidate.Name == name {
			return
		}
	}
	if task, err := source.Task(name); err == nil {
		t.Tasks = append(t.Tasks, task)
	}
}

// Task returns a task for supplied name
func (t *TasksNode) Task(name string) (*Task, error) {
	if len(t.Tasks) == 0 {
		return nil, fmt.Errorf("failed to LookupValueNode task: %v", name)
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
	return nil, fmt.Errorf("failed to LookupValueNode task: %v", name)
}

// Task returns a task for supplied name
func (t *TasksNode) Has(name string) bool {
	if len(t.Tasks) == 0 {
		return false
	}
	_, err := t.Task(name)
	return err == nil
}

func (t *TasksNode) Clone() *TasksNode {
	ret := *t
	return &ret
}
