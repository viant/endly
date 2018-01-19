package main

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"log"
	"path"
)

//main generate file with static content from meta, workflow, req folders so that they can be compiled into final binary
func main() {
	genDirectory := toolbox.CallerDirectory(3)
	parent := string(genDirectory[:len(genDirectory)-4])
	destinationRoot := "github.com/viant/endly/"
	mappings := []*storage.StorageMapping{
		{
			SourceURL:      toolbox.FileSchema + path.Join(parent, "meta"),
			DestinationURI: path.Join(destinationRoot, "meta"),
			TargetFile:     path.Join(parent, "static", "meta.go"),
			TargetPackage:  "static",
		},
		{
			SourceURL:      toolbox.FileSchema + path.Join(parent, "workflow"),
			DestinationURI: path.Join(destinationRoot, "workflow"),
			TargetFile:     path.Join(parent, "static", "workflow.go"),
			TargetPackage:  "static",
		},
		{
			SourceURL:      toolbox.FileSchema + path.Join(parent, "req"),
			DestinationURI: path.Join(destinationRoot, "req"),
			TargetFile:     path.Join(parent, "static", "req.go"),
			TargetPackage:  "static",
		},
	}
	err := storage.GenerateStorageCode(mappings...)
	if err != nil {
		log.Fatal(err)
	}
}
