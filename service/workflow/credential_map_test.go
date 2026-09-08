package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/credential"
)

func TestApplyCredentialMap_rootRegisters(t *testing.T) {
	manager := endly.New()
	context := manager.NewContext(nil)
	svc := context.Secrets.(*credential.Service)

	request := &RunRequest{
		Inlined: &model.Inlined{
			CredentialMap: map[string]string{
				"gcp-e2e": "op://Private/gcp-e2e.json/notesPlain",
			},
		},
		AssetURL: "file:///tmp/run.yaml",
	}
	wf := &model.Workflow{Source: location.NewResource("file:///tmp/run.yaml")}
	require.NoError(t, applyCredentialMap(context, request, wf, true))
	require.True(t, svc.HasCredentialMap())

	resolved, err := svc.ResolveAlias("gcp-e2e")
	require.NoError(t, err)
	require.Equal(t, "op://Private/gcp-e2e.json/notesPlain", resolved)
}

func TestApplyCredentialMap_childWithMapErrors(t *testing.T) {
	manager := endly.New()
	context := manager.NewContext(nil)

	request := &RunRequest{
		Inlined: &model.Inlined{
			CredentialMap: map[string]string{
				"gcp-e2e": "op://Private/gcp-e2e.json/notesPlain",
			},
		},
		AssetURL: "file:///tmp/child.yaml",
	}
	wf := &model.Workflow{Source: location.NewResource("file:///tmp/child.yaml")}
	err := applyCredentialMap(context, request, wf, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "root workflow")
	require.Contains(t, err.Error(), "child.yaml")
}

func TestApplyCredentialMap_childWithoutMapOK(t *testing.T) {
	manager := endly.New()
	context := manager.NewContext(nil)
	request := &RunRequest{AssetURL: "file:///tmp/child.yaml"}
	wf := &model.Workflow{}
	require.NoError(t, applyCredentialMap(context, request, wf, false))
}

func TestApplyCredentialMap_fromWorkflowField(t *testing.T) {
	manager := endly.New()
	context := manager.NewContext(nil)
	svc := context.Secrets.(*credential.Service)

	request := &RunRequest{AssetURL: "file:///tmp/run.yaml"}
	wf := &model.Workflow{
		Source: location.NewResource("file:///tmp/run.yaml"),
		CredentialMap: map[string]string{
			"gcp-e2e": "/tmp/sa.json",
		},
	}
	require.NoError(t, applyCredentialMap(context, request, wf, true))
	resolved, err := svc.ResolveAlias("gcp-e2e")
	require.NoError(t, err)
	require.Equal(t, "/tmp/sa.json", resolved)
}
