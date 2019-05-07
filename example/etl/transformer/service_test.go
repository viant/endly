package transformer_test

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/viant/asc"

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

	var transformedBaseDirectory = path.Join(toolbox.CallerDirectory(3), "test/transformed")
	toolbox.CreateDirIfNotExist(transformedBaseDirectory)

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
			Columns:   []string{"customer_id", "visits_date", "visits_count"},
			PkColumns: []string{"customer_id"},
		},
		Transformer: "Flatten",
	})

	assert.Equal(t, "", response.Error)
}

//
//
//
//func TestService_CopyWithAerospikFlatte(t *testing.T) {
//	service := transformer.NewService()
//	var baseDirectory = path.Join(toolbox.CallerDirectory(3), "test")
//	toolbox.RemoveFileIfExist(path.Join(baseDirectory, "transformed/users.json"))
//	//	defer toolbox.RemoveFileIfExist(path.Join(baseDirectory, "transformed/users.json"))
//
//
//	sourceConfig := &dsc.Config{DriverName:"aerospike",
//		Descriptor:"[url]",
//		Parameters:map[string]string{
//			"dbname": "db4",
//			"namespace": "db4",
//			"host": "127.0.0.1",
//			"port": "3000",
//			"dateFormat": "yyyy-MM-dd hh:mm:ss",
//			"keyColumnName": "id",
//			"optimizeLargeScan": "true",
//			"inheritIdFromPK":"false",
//		},
//	}
//	targetConfig := &dsc.Config{
//		DriverName:"mysql",
//		Descriptor: "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true",
//		Parameters:map[string]string {
//			"password": "dev",
//			"dbname":"db3",
//			"username": "dev",
//		},
//	}
//
//
//
//	response := service.Copy(&transformer.CopyRequest{
//		InsertMode: true,
//		MaxThreads:  2,
//		Source: &transformer.DatasetResource{
//			DsConfig: sourceConfig,
//			Table:    "users",
//			SQL:      "SELECT id AS user_id, visits AS visit FROM users",
//		},
//		Destination: &transformer.DatasetResource{
//			DsConfig:  targetConfig,
//			Table: "user_visits",
//			PkColumns:[]string{"visit_date", "visit_app_id","user_id"},
//			Columns:[]string{"user_id","visit_date", "visit_count", "visit_app_id"},
//		},
//		Transformer:"Flatten",
//	})
//
//	assert.Equal(t, "", response.Error)
//}
