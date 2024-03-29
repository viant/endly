package shared

import (
	"context"
	"embed"
	"github.com/viant/afs"
	_ "github.com/viant/afs/embed"
	"github.com/viant/afs/option"
	"strings"
)

//go:embed endly/*
var embedFs embed.FS
func init() {
	fs := afs.New()
	objects, _ := fs.List(context.Background(), "embed:///endly", &embedFs, option.NewRecursive(true))
	if len(objects) == 0 {
		return
	}
	for _, object := range objects {
		if object.IsDir() {
			continue
		}
		data, err := fs.DownloadWithURL(context.Background(), object.URL(), &embedFs)
		if err != nil {
			continue
		}
		URL := strings.Replace(strings.Replace(object.URL(), "embed://", "mem://", 1), "localhost", "github.com/viant", 1)
		_= fs.Upload(context.Background(), URL, 0644, strings.NewReader(string(data)))
	}
}
