package endly_test

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/viant/dsc"
	"github.com/viant/dsunit"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"os"
	"path"
	"testing"
)

func TestDsUnitService(t *testing.T) {

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
			Scripts: []*endly.Resource{
				endly.NewFileResource("test/dsunit/mydb1.sql"),
			},
		})

		assert.Equal(t, "", response.Error)
		response = service.Run(context, &endly.DsUnitPrepareRequest{
			Datastore: "mydb1",
			Prefix:    "prepare_",
			URL:       endly.NewFileResource("test/dsunit/dataset1").URL,
		})
		assert.Equal(t, "", response.Error)

		response = service.Run(context, &endly.DsUnitVerifyRequest{
			Datasets: &dsunit.DatasetResource{
				Datastore: "mydb1",
				Prefix:    "verify_",
				URL:       endly.NewFileResource("test/dsunit/dataset1").URL,
			},
		})
		assert.Equal(t, "", response.Error)
		verifyResponse, ok := response.Response.(*endly.DsUnitVerifyResponse)
		assert.True(t, ok)
		assert.Equal(t, 2, verifyResponse.DatasetChecked["ACCOUNT"])
		assert.Equal(t, 2, verifyResponse.DatasetChecked["USER"])

		response = service.Run(context, &endly.DsUnitVerifyRequest{
			Datasets: &dsunit.DatasetResource{
				Datastore: "mydb1",
				Prefix:    "err_",
				URL:       endly.NewFileResource("test/dsunit/dataset1").URL,
			},
		})
		assert.True(t, response.Error != "")

		response = service.Run(context, &endly.DsUnitMappingRequest{
			Mappings: []*endly.Resource{

				endly.NewFileResource("test/workflow/mapping.json"),
			},
		})
		assert.Equal(t, "", response.Error)
		mappingResponse, ok := response.Response.(*endly.DsUnitMappingResonse)
		if assert.True(t, ok) {
			assert.Equal(t, []string{"USER", "ACCOUNT"}, mappingResponse.Tables)

		}

		response = service.Run(context, &endly.DsUnitTableSequenceRequest{
			Datastore: "mydb1",
			Tables:    []string{"USER", "ACCOUNT"},
		})

		assert.Equal(t, "", response.Error)
		sequenceResponse, ok := response.Response.(*endly.DsUnitTableSequenceResponse)
		if assert.True(t, ok) {
			assert.Equal(t, map[string]int{
				"USER":    4,
				"ACCOUNT": 4,
			}, sequenceResponse.Sequences)

		}
		lastUserId := sequenceResponse.Sequences["USER"]
		lastAccountId := sequenceResponse.Sequences["ACCOUNT"]

		response = service.Run(context, &endly.DsUnitPrepareRequest{
			Datastore: "mydb1",
			Data: map[string][]map[string]interface{}{
				"USER_ACCOUNT": {
					{
						"USER_ID":    lastUserId,
						"ACCOUNT_ID": lastAccountId,
						"NAME":       "TestUser",
						"TYPE":       "Testtype",
						"EMAIL":      "a2@wrwe.pl",
					},
					{
						"USER_ID":    lastUserId + 1,
						"ACCOUNT_ID": lastAccountId,
						"EMAIL":      "a3@wrwe.pl",
					},
				},
			},
		})
		assert.Equal(t, "", response.Error)
		prepareResponse, ok := response.Response.(*endly.DsUnitPrepareResponse)
		if assert.True(t, ok) {
			assert.Equal(t, 3, prepareResponse.Added)
			assert.Equal(t, 0, prepareResponse.Modified)

		}

	}

	//Test running dsunit vi workflow
	manager, service, err = getServiceWithWorkflow("test/workflow/dsunit_workflow.csv")
	if !assert.Nil(t, err) {
		return
	}
	assert.NotNil(t, manager)
	assert.NotNil(t, service)

	{
		context := manager.NewContext(toolbox.NewContext())
		response := service.Run(context, &endly.WorkflowRunRequest{
			Name:   "dsunit",
			Params: map[string]interface{}{},
		})
		assert.Equal(t, "", response.Error)
		serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
		assert.True(t, ok)
		assert.NotNil(t, serviceResponse)

	}

}
