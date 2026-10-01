package model

import "testing"

func TestPathSelectionOrderAndAncestors(t *testing.T) {
	a := NewTask("a", false)
	a.Tasks = Tasks{NewTask("x", false), NewTask("y", false)}
	a.Init = Variables{{Name: "parent", Value: "yes"}}
	b := NewTask("b", false)
	root := &TasksNode{Tasks: Tasks{a, b}}
	selected, err := root.SelectWithMode(TasksSelector("a.y,b,a.x,a.y"), "path")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Tasks) != 4 {
		t.Fatal("selections were merged or dropped")
	}
	for i, name := range []string{"a", "b", "a", "a"} {
		if selected.Tasks[i].Name != name {
			t.Fatalf("position %d: %s", i, selected.Tasks[i].Name)
		}
	}
	if selected.Tasks[0].Tasks[0].Name != "y" || selected.Tasks[2].Tasks[0].Name != "x" {
		t.Fatal("wrong child")
	}
	if len(a.Tasks) != 2 || len(selected.Tasks[0].Init) != 1 {
		t.Fatal("source mutated or ancestor state lost")
	}
	for _, selector := range []string{"x", "a.missing", "a..x", "a.x."} {
		if _, err := root.SelectWithMode(TasksSelector(selector), "path"); err == nil {
			t.Fatalf("accepted %s", selector)
		}
	}
	if _, err := root.SelectWithMode("*", "unknown"); err == nil {
		t.Fatal("accepted invalid mode")
	}
}

func TestPathSelectionControlTasks(t *testing.T) {
	a := NewTask("a", false)
	a.TasksNode = &TasksNode{Tasks: Tasks{NewTask("x", false), NewTask("cleanup", false)}, DeferredTask: "cleanup"}
	root := &TasksNode{Tasks: Tasks{a, NewTask("final", false)}, DeferredTask: "final"}
	selected, err := root.SelectWithMode("a.x", "path")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Tasks) != 2 || len(selected.Tasks[0].Tasks) != 2 {
		t.Fatal("control tasks lost")
	}
}
