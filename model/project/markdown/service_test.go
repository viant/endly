package markdown

import (
	"context"
	"embed"
	"fmt"
	_ "github.com/viant/afs/embed"

	"github.com/stretchr/testify/assert"
	"github.com/viant/endly/model/project/loader"
	"github.com/viant/endly/model/project/option"
	"testing"
)

//go:embed testdata/*
var embedFS embed.FS

func TestNew(t *testing.T) {

	srv := New()
	//_, err := srv.Load(context.Background(), "embed:///testdata/example.md", option.WithEmbedFS(&embedFS))
	//if ! assert.Nil(t, err) {
	//	return
	//}

	service := loader.New()
	bundle, err := service.Load(context.Background(), "embed:///testdata/projectx/run.yaml",
		option.WithDependencies(true),
		option.WithAssets(true),
		option.WithEmbedFS(&embedFS),
	)
	if !assert.Nil(t, err) {
		return
	}
	data, err := srv.Markdown(context.Background(), bundle.Workflow, option.WithDependencies(true), option.WithAssets(true))
	if !assert.Nil(t, err) {
		return
	}
	fmt.Printf("%s\n", data)
}
