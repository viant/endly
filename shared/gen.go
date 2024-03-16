package shared

import (
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"log"
	"path"

	"strings"
)

// main generate file with static content from meta, workflow, req folders so that they can be compiled into final binary
func main() {
	callerDir := toolbox.CallerDirectory(3)
	parent := strings.Replace(callerDir, "/shared/", "", 1)

	mappings := []*storage.StorageMapping{
		{
			SourceURL:      toolbox.FileSchema + path.Join(parent, "shared/meta"),
			DestinationURI: path.Join(endly.Namespace, "meta"),
			TargetFile:     path.Join(parent, "shared/static", "meta.go"),
			TargetPackage:  "static",
		},
		{
			SourceURL:      toolbox.FileSchema + path.Join(parent, "shared/workflow"),
			DestinationURI: path.Join(endly.Namespace, "workflow"),
			TargetFile:     path.Join(parent, "shared/static", "workflow.go"),
			TargetPackage:  "static",
		},

		{
			SourceURL:      toolbox.FileSchema + path.Join(parent, "Version"),
			DestinationURI: path.Join(endly.Namespace, "Version"),
			TargetFile:     path.Join(parent, "shared/static", "version.go"),
			TargetPackage:  "static",
		},
	}
	err := storage.GenerateStorageCode(mappings...)
	if err != nil {
		log.Fatal(err)
	}
}
