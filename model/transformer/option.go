package transformer

import "embed"

type Option func(o *Options)

func WithDependencies() Option {
	return func(o *Options) {
		o.WithDependencies = true
	}
}

func WithAssets() Option {
	return func(o *Options) {
		o.WithAssets = true
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
