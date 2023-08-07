package main

import (
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"log"
	"path"
)

// main generate file with static from asset and template into memory storage
func main() {
	parent := toolbox.CallerDirectory(3)
	mappings := []*storage.StorageMapping{
		{
			SourceURL:      toolbox.FileSchema + path.Join(parent, "asset"),
			DestinationURI: path.Join(endly.Namespace, "asset"),
			TargetFile:     path.Join(parent, "static", "asset.go"),
			TargetPackage:  "static",
			UseTextFormat:  true,
		},
		{
			SourceURL:      toolbox.FileSchema + path.Join(parent, "template"),
			DestinationURI: path.Join(endly.Namespace, "template"),
			TargetFile:     path.Join(parent, "static", "template.go"),
			TargetPackage:  "static",
			UseTextFormat:  true,
		},
	}
	err := storage.GenerateStorageCode(mappings...)
	if err != nil {
		log.Fatal(err)
	}
}
