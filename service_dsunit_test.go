package endly_test

import (
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/stretchr/testify/assert"
	"testing"
	_ "github.com/go-sql-driver/mysql"
	"github.com/viant/dsc"
	"path"
	"os"
	"github.com/viant/dsunit"
)

func TestSdUnitService(t *testing.T) {



	manager := endly.NewManager()

	context := manager.NewContext(toolbox.NewContext())
	service, err := manager.Service(endly.DataStoreUnitServiceId)
	assert.Nil(t, err)
	assert.NotNil(t, service)



	if toolbox.FileExists(path.Join(os.Getenv("HOME"), "secret/mysql.json")) {


		response := service.Run(context, &endly.DsUnitRegisterRequest{
			Datastore: "mydb1",
			Config: &dsc.Config{
				DriverName: "mysql",
				Descriptor: "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true",
			},
			Credential:     path.Join(os.Getenv("HOME"), "secret/mysql.json"),
			AdminDatastore: "mysql",
			ClearDatastore: true,
			Schema:         endly.NewFileResource("test/dsunit/mydb1.sql"),
		})
		assert.Equal(t, "", response.Error)

		response = service.Run(context, &endly.DsUnitPrepareRequest{
			Datasets:&dsunit.DatasetResource{
				Datastore:"mydb1",
				Prefix:"prepare_",
				URL:endly.NewFileResource("test/dsunit/dataset1").URL,
			},

		})
		assert.Equal(t, "", response.Error)


		response = service.Run(context, &endly.DsUnitVerifyRequest{
			Datasets:&dsunit.DatasetResource{
				Datastore:"mydb1",
				Prefix:"verify_",
				URL:endly.NewFileResource("test/dsunit/dataset1").URL,
			},

		})
		assert.Equal(t, "", response.Error)
		verifyResponse, ok := response.Response.(*endly.DsUnitVerifyResponse)
		assert.True(t, ok)
		assert.Equal(t, 2, verifyResponse.DatasetChecked["ACCOUNT"])
		assert.Equal(t, 2, verifyResponse.DatasetChecked["USER"])

	}




}
