package dsunit

import (
	_ "github.com/mattn/go-sqlite3"
	"github.com/viant/endly"
)

func init() {
	endly.Registry.Register(func() endly.Service {
		service := New()
		return service
	})
	endly.PredefinedUdfs["AsTableRecords"] = AsTableRecords
}
