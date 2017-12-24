package transformer_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/dsc"
	"github.com/viant/endly/example/etl/transformer"
	"github.com/viant/toolbox"
	"path"
	"testing"
)

func TestService_Copy(t *testing.T) {
	service := transformer.NewService()
	var baseDirectory = path.Join(toolbox.CallerDirectory(3), "test")
	toolbox.RemoveFileIfExist(path.Join(baseDirectory, "transformed/apps_keys.json"))
	defer toolbox.RemoveFileIfExist(path.Join(baseDirectory, "transformed/apps_keys.json"))
	sourceConfig := dsc.NewConfig("csv", "[url]", "dateFormat:yyyy-MM-dd hh:mm:ss,ext:csv,url:"+"file://"+path.Join(baseDirectory, "data"))
	targetConfig := dsc.NewConfig("ndjson", "[url]", "dateFormat:yyyy-MM-dd hh:mm:ss,ext:json,url:"+"file://"+path.Join(baseDirectory, "transformed"))

	response := service.Copy(&transformer.CopyRequest{
		InsertMode: true,
		BatchSize:  2,
		Source: &transformer.DatasetResource{
			DsConfig: sourceConfig,
			Table:    "apps",
			SQL:      "SELECT * FROM apps",
		},
		Destination: &transformer.DatasetResource{
			DsConfig:  targetConfig,
			Table:     "apps_keys",
			PkColumns: []string{"APP_ID"},
		},
	})
	assert.Equal(t, "", response.Error)
	assert.Equal(t, "", response.Error)
}

func TestService_CopyWithTransformer(t *testing.T) {
	service := transformer.NewService()
	var baseDirectory = path.Join(toolbox.CallerDirectory(3), "test")
	toolbox.RemoveFileIfExist(path.Join(baseDirectory, "transformed/visits.json"))
	defer toolbox.RemoveFileIfExist(path.Join(baseDirectory, "transformed/visits.json"))

	sourceConfig := dsc.NewConfig("ndjson", "[url]", "dateFormat:yyyy-MM-dd hh:mm:ss,ext:json,url:"+"file://"+path.Join(baseDirectory, "data"))
	targetConfig := dsc.NewConfig("ndjson", "[url]", "dateFormat:yyyy-MM-dd hh:mm:ss,ext:json,url:"+"file://"+path.Join(baseDirectory, "transformed"))

	response := service.Copy(&transformer.CopyRequest{
		InsertMode: true,
		BatchSize:  2,
		Source: &transformer.DatasetResource{
			DsConfig: sourceConfig,
			Table:    "visits",
			SQL:      "SELECT * FROM visits",
		},
		Destination: &transformer.DatasetResource{
			DsConfig:  targetConfig,
			Table:     "visits",
			Columns:   []string{"customer_id", "visits"},
			PkColumns: []string{"customer_id"},
		},
		Transformer: "MapToSlice",
	})

	assert.Equal(t, "", response.Error)
}
