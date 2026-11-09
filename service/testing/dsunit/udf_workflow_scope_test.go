package dsunit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	_ "github.com/viant/endly/service/testing/dsunit"
	"github.com/viant/endly/service/workflow"
)

func TestAsTableRecordsPublishesAliasesForTemplateInvocation(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "cases", "001_member", "dbsetup")
	require.NoError(t, os.MkdirAll(fixture, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(fixture, "APP_USERS.json"), []byte(`[{"ID":"$Sequences.APP_USERS/${tag}.member","NAME":"member"}]`), 0600))
	source := `init:
  app_db: {}
  observed: {}
  Sequences:
    APP_USERS: 100
pipeline:
  populate:
    action: run
    request: "@populate"
  cases:
    tag: $pathMatch
    data:
      '[]app_db_dbsetup': '@dbsetup'
    subPath: cases/${index}_*
    template:
      check:
        action: nop
        init:
          member: ${app_db.APP_USERS.${tag}.member}
          observed.ID: ${member.ID}
`
	path := filepath.Join(dir, "run.yaml")
	require.NoError(t, os.WriteFile(path, []byte(source), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "populate.yaml"), []byte("pipeline:\n  convert:\n    action: nop\n    init:\n      rows: $AsTableRecords('data.app_db_dbsetup/app_db')\n"), 0600))
	manager := endly.New()
	manager.Register(workflow.New())
	ctx := manager.NewContext(nil)
	defer ctx.Close()
	var response workflow.RunResponse
	request, err := workflow.NewRunRequestFromURL(path)
	require.NoError(t, err)
	request.AssetURL, request.Name, request.SharedState = path, "run", true
	require.NoError(t, endly.Run(ctx, request, &response))
	state := ctx.State()
	actual, _ := state.GetValue("observed.ID")
	require.EqualValues(t, 100, actual)
}
