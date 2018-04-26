package static

import (
	"github.com/viant/toolbox/storage"
)

func Start() {
	storage.NewMemoryService()
}
