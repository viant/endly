package static

import (
	"fmt"
	"github.com/viant/toolbox/storage"
)

func Start() {
	var memStorage = storage.NewMemoryService()
	fmt.Printf("uses :%T\n", memStorage)
}
