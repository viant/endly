package static

import (
	"bytes"
	"github.com/viant/toolbox/storage"
	"log"
)

func init() {
	var memStorage = storage.NewMemoryService()
	{
		err := memStorage.Upload("mem://github.com/viant/endly/Version", bytes.NewReader([]byte{48, 46, 51, 55, 46, 52}))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/Version %v", err)
		}
	}
}
