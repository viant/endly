package transfer

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/viant/afs"
	_ "github.com/viant/afs/embed"
	"github.com/viant/afs/file"
	"github.com/viant/afs/url"
	"github.com/viant/endly/model/transformer"
	"testing"
)

//go:embed testdata/*
var embedFS embed.FS

func TestNew(t *testing.T) {
	service := New()
	assert.NotNil(t, service)
	bundle, err := service.Transfer(context.Background(), "embed:///testdata/projectx/run.yaml",
		transformer.WithDependencies(true),
		transformer.WithAssets(true),
		transformer.WithEmbedFS(&embedFS),
	)
	assert.Nil(t, err)
	assert.NotNil(t, bundle)

	if err := upload("PROJECT.json", bundle.Projects());err != nil {
		panic(err)
	}
	if err := upload("WORKFLOW.json", bundle.Workflows());err != nil {
		panic(err)
	}
	if err := upload("TASK.json", bundle.AllTasks());err != nil {
		panic(err)
	}

	if err := upload("ASSET.json", bundle.AllAssets());err != nil {
		panic(err)
	}


}


func upload(URI string, data interface{}) error {
	baseURL := "/Users/awitas/go/src/github.com/viant/endly/model/transfer/datly/e2e/database/mysql/endly/static"
	payload, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	fs := afs.New()
	return fs.Upload(context.Background(), url.Join(baseURL, URI),  file.DefaultFileOsMode, bytes.NewReader(payload))
}