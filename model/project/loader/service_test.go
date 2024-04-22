package loader

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/afs"
	_ "github.com/viant/afs/embed"
	"github.com/viant/afs/file"
	"github.com/viant/afs/url"
	option "github.com/viant/endly/model/project/option"
	"gopkg.in/yaml.v3"
	"testing"
)

var uploadAsset = false

//go:embed testdata/*
var embedFS embed.FS

func TestNew(t *testing.T) {
	service := New()
	assert.NotNil(t, service)
	bundle, err := service.Load(context.Background(), "embed:///testdata/projectx/run.yaml",
		option.WithDependencies(true),
		option.WithAssets(true),
		option.WithEmbedFS(&embedFS),
	)
	assert.Nil(t, err)
	assert.NotNil(t, bundle)

	if err := upload("PROJECT.json", bundle.Projects()); err != nil {
		panic(err)
	}
	if err := upload("WORKFLOW.json", bundle.Workflows()); err != nil {
		panic(err)
	}
	if err := upload("TASK.json", bundle.Tasks()); err != nil {
		panic(err)
	}

	if err := upload("ASSET.json", bundle.Assets()); err != nil {
		panic(err)
	}

	for _, w := range bundle.Workflows() {
		fmt.Printf("Workflow: %v\n", w.Name)
		data, err := yaml.Marshal(w)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Workflow: %s\n", data)
	}
}

func upload(URI string, data interface{}) error {

	if !uploadAsset {
		return nil
	}
	baseURL := "model/project/dao/e2e/database/mysql/endly/static"
	payload, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	fs := afs.New()
	return fs.Upload(context.Background(), url.Join(baseURL, URI), file.DefaultFileOsMode, bytes.NewReader(payload))
}
