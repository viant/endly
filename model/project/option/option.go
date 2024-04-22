package option

import (
	"embed"
	"github.com/viant/endly/model/graph"
)

type Option func(o *Options)

func WithDependencies(flag bool) Option {
	return func(o *Options) {
		o.WithDependencies = flag
	}
}

func WithAssets(flag bool) Option {
	return func(o *Options) {
		o.WithAssets = flag
	}
}

func WithEmbedFS(embedFS *embed.FS) Option {
	return func(o *Options) {
		o.EmbedFS = embedFS
	}
}

func WithURI(uri string) Option {
	return func(o *Options) {
		o.URI = uri
	}
}

func WithProjectID(projectID string) Option {
	return func(o *Options) {
		o.ProjectID = projectID
	}
}

func WithTemplate(template string) Option {
	return func(o *Options) {
		o.Template = template
	}
}

func WithBaseURL(baseURL string) Option {
	return func(o *Options) {
		o.BaseURL = baseURL
	}
}

func WithIsRoot(flag bool) Option {
	return func(o *Options) {
		o.Root = &flag
	}
}

func WithAssetsManager(assets *graph.AssetManager) Option {
	return func(o *Options) {
		o.Assets = assets
	}
}

func WithParentWorkflowID(parentWorkflowID string) Option {
	return func(o *Options) {
		o.ParentWorkflowID = parentWorkflowID
	}
}

func WithInstance(instance *graph.Instance) Option {
	return func(o *Options) {
		o.Instance = instance
	}
}
