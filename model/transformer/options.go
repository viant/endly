package transformer

import (
	"embed"
	"github.com/viant/afs/storage"
)

type Options struct {
	WithDependencies bool
	WithAssets       bool
	EmbedFS          *embed.FS
	opts             []Option
	ProjectID        string
	URI              string
	Template         string
}

var emptyOptions = []storage.Option{}

func (o *Options) Options(opts ...Option) []Option {
	return append(o.opts, opts...)
}

func (o *Options) StorageOptions() []storage.Option {
	if o.EmbedFS != nil {
		return []storage.Option{o.EmbedFS}
	}
	return emptyOptions
}

func NewOptions(opts ...Option) *Options {
	ret := &Options{opts: opts}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
}
