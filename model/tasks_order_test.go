package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTasksNodeSelectPreservesOrderAndDuplicates(t *testing.T) {
	node := &TasksNode{Tasks: []*Task{
		NewTask("task1", false),
		NewTask("task2", false),
		NewTask("task3", false),
	}}
	selected := node.Select(TasksSelector("task1,task2,task1"))
	require.Equal(t, []string{"task1", "task2", "task1"}, selectedTaskNames(selected.Tasks))
}

func TestTasksNodeSelectKeepsControlTasksWithoutReorderingSelection(t *testing.T) {
	node := &TasksNode{
		Tasks: []*Task{
			NewTask("task1", false),
			NewTask("task2", false),
			NewTask("catch", false),
			NewTask("defer", false),
		},
		OnErrorTask:  "catch",
		DeferredTask: "defer",
	}
	selected := node.Select(TasksSelector("task2,task1,task2"))
	require.Equal(t, []string{"task2", "task1", "task2", "catch", "defer"}, selectedTaskNames(selected.Tasks))
}

func selectedTaskNames(tasks Tasks) []string {
	result := make([]string, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, task.Name)
	}
	return result
}
