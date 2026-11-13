package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskLookupPrefersOwningScope(t *testing.T) {
	for _, name := range []string{"catch", "defer", "run"} {
		t.Run(name, func(t *testing.T) {
			nested := NewTask(name, false)
			child := NewTask("child", false)
			child.TasksNode = &TasksNode{Tasks: []*Task{nested}}
			local := NewTask(name, false)
			root := &TasksNode{Tasks: []*Task{child, local}}
			got, err := root.Task(name)
			require.NoError(t, err)
			require.Same(t, local, got)
			root.Tasks = root.Tasks[:1]
			got, err = root.Task(name)
			require.NoError(t, err)
			require.Same(t, nested, got, "retain descendant fallback when no local task exists")
		})
	}
}

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
