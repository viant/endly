package bigquery

import (
	"github.com/viant/endly"
)

func init() {
	endly.PredefinedUdfs["BigQueryRecords"] = QueryRecords
	_ = endly.Registry.Register(func() endly.Service {
		return New()
	})

}
