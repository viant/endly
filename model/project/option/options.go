package option

import (
	"embed"
	"github.com/viant/afs/storage"
	"github.com/viant/endly/model/graph"
)

type Options struct {
	WithDependencies bool
	WithAssets       bool
	EmbedFS          *embed.FS
	opts             []Option
	ProjectID        string
	URI              string
	Template         string
	BaseURL          string
	ParentWorkflowID string
	Instance         *graph.Instance
	Assets           *graph.AssetManager
	Root             *bool
}

func (o *Options) IsRoot() bool {
	if o.Root == nil {
		return false
	}
	return *o.Root
}

func (o *Options) Options(opts ...Option) []Option {
	return append(o.opts, opts...)
}

func (o *Options) StorageOptions(opts ...storage.Option) []storage.Option {
	if o.EmbedFS != nil {
		return append(opts, o.EmbedFS)
	}
	return opts
}

func NewOptions(opts ...Option) *Options {
	ret := &Options{opts: opts}
	for _, opt := range opts {
		opt(ret)
	}
	if ret.Root == nil {
		t := true
		ret.Root = &t
	}
	return ret
}
