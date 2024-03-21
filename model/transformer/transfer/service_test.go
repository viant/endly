package transfer

import (
	"context"
	"embed"
	"github.com/stretchr/testify/assert"
	_ "github.com/viant/afs/embed"
	"github.com/viant/endly/model/transformer"
	"testing"
)

//go:embed testdata/*
var embedFS embed.FS

func TestNew(t *testing.T) {
	service := New()
	assert.NotNil(t, service)
	bundle, err := service.Transfer(context.Background(), "embed:///testdata/projectx/run.yaml",
		transformer.WithDependencies(),
		transformer.WithAssets(),
		transformer.WithEmbedFS(&embedFS),
	)
	assert.Nil(t, err)
	assert.NotNil(t, bundle)
}
