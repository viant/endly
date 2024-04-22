package graph

import (
	"context"
	"embed"
	"github.com/stretchr/testify/assert"
	_ "github.com/viant/afs/embed"
	"testing"
)

//go:embed testdata/*
var embeddFs embed.FS

func TestService_LoadWorkflow(t *testing.T) {
	srv := New()
	URL := "embed:///testdata/appx_regression.yaml"
	workflow, err := srv.LoadWorkflow(context.Background(), URL, &embeddFs)
	if !assert.Nil(t, err) {
		return
	}
	if !assert.NotNil(t, workflow) {
		return
	}
	assert.Equal(t, "appx_regression", workflow.Name)

	var tasks []string
	workflow.Tasks(func(name string, task *Node) error {
		tasks = append(tasks, name)
		return nil
	})
	assert.Equal(t, []string{"updateArch", "init", "test", "sleep", "validateLogs"}, tasks)
	test, err := workflow.Task("test")
	if !assert.Nil(t, err) {
		return
	}
	assert.Equal(t, "test", test.Name)
	rng, ok := test.String("range")
	assert.True(t, ok)
	assert.Equal(t, "1..782", rng)
	tmplData := test.Data()
	assert.Equal(t, 352, len(tmplData))
	var templateTask []string
	template := test.Template()
	assert.NotNil(t, template)
	err = template.Tasks(func(name string, task *Node) error {
		templateTask = append(templateTask, name)
		return nil
	})
	if !assert.Nil(t, err) {
		return
	}
	assert.Equal(t, []string{"checkSkip", "test"}, templateTask)
}
