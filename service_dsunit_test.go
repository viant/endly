package endly_test

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/viant/dsc"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"os/exec"
	"path"
	"testing"
	"github.com/viant/dsunit"
	"errors"
	"github.com/viant/toolbox/data"
)

func getRegisteredDsUnitService(manager endly.Manager, context *endly.Context, dbname string) (endly.Service, error) {
	var baseDir = "/tmp/test/endly/dsunit/"
	exec.Command("rm", "-rf", baseDir)
	toolbox.CreateDirIfNotExist(baseDir)
	credential, err := GetDummyCredential()
	if err != nil {
		return nil, err
	}
	service, err := manager.Service(endly.DataStoreUnitServiceID)
	if err != nil {
		return nil, err
	}

	config, err := dsc.NewConfigWithParameters("sqlite3", "[url]", credential, map[string]interface{}{
		"url": path.Join(baseDir, dbname),
	})

	if err != nil {
		return nil, err
	}
	response := service.Run(context, dsunit.NewInitRequest(
		dbname,
		true,
		dsunit.NewRegisterRequest(dbname, config),
		nil,
		nil,
		dsunit.NewRunScriptRequest(dbname, url.NewResource(fmt.Sprintf("test/dsunit/%v.sql", dbname)))))

	if response.Error != "" {
		return nil, errors.New(response.Error)
	}
	return service, nil
}

func TestDsUnitService(t *testing.T) {

	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	service, err := getRegisteredDsUnitService(manager, context, "mydb1")
	if assert.Nil(t, err) {
		serviceResponse := service.Run(context, dsunit.NewPrepareRequest(dsunit.NewDatasetResource("mydb1", url.NewResource("test/dsunit/dataset1").URL, "prepare_", "")))
		assert.Equal(t, "", serviceResponse.Error)

		serviceResponse = service.Run(context, dsunit.NewExpectRequest(0,
			dsunit.NewDatasetResource("mydb1", url.NewResource("test/dsunit/dataset1").URL, "verify_", "")))

		assert.Equal(t, "", serviceResponse.Error)
		verifyResponse, ok := serviceResponse.Response.(*dsunit.ExpectResponse)
		assert.True(t, ok)
		assert.EqualValues(t, 0, len(verifyResponse.Validation[0].Failures))


			serviceResponse = service.Run(context, &dsunit.MappingRequest{
				Mappings: []*dsunit.Mapping{
					{
						Resource: url.NewResource("test/dsunit/user_account.json"),
					},

				},
			})
			assert.Equal(t, "", serviceResponse.Error)

			var tables []string
			{
				response, ok := serviceResponse.Response.(*dsunit.MappingResponse)
				if assert.True(t, ok) {
					assert.EqualValues(t, []string{"ACCOUNT", "USER"}, response.Tables)
				}
				tables = response.Tables
			}
			serviceResponse = service.Run(context, &dsunit.SequenceRequest{
				Datastore: "mydb1",
				Tables:    tables,
			})
			var sequences map[string]int
			{
				response, ok := serviceResponse.Response.(*dsunit.SequenceResponse)
				if assert.True(t, ok) {
					assert.EqualValues(t, map[string]int{
						"USER":    3,
						"ACCOUNT": 3,
					}, response.Sequences)
				}
				sequences = response.Sequences
			}
			assert.NotNil(t, sequences)

			serviceResponse = service.Run(context, &dsunit.RunScriptRequest{
				Datastore: "mydb1",
				Scripts: []*url.Resource{
					url.NewResource("test/dsunit/mydb1.sql"),
				},
			})
			{
				response, ok := serviceResponse.Response.(*dsunit.RunSQLResponse)
				if assert.True(t, ok) {
					assert.EqualValues(t, 0, response.RowsAffected)
				}
			}

			for k, v := range sequences {
				sequences[k] = v + 1
			}

			var tableData = make([]*endly.DsUnitTableData, 0)
			tableData = append(tableData, &endly.DsUnitTableData{
				Table: "USER_ACCOUNT",
				Value: map[string]interface{}{
					"ACCOUNT_ID":   "$meta.ACCOUNT_ID",
					"USER_ID":      "$meta.USER_ID",
					"NAME":         "Bob",
					"TYPE":         "direct",
					"CONTACT_ID":   10,
					"CONTACT_NAME": "Smith",
					"EMAIL":        "bob@email.com",
				},
				PostIncrement: []string{"meta.ACCOUNT_ID", "meta.USER_ID"},
			},
				&endly.DsUnitTableData{
					Table: "USER_ACCOUNT",
					Value: []interface{}{map[string]interface{}{
						"ACCOUNT_ID":   "$meta.ACCOUNT_ID",
						"USER_ID":      "$meta.USER_ID",
						"NAME":         "Rober",
						"TYPE":         "direct",
						"CONTACT_ID":   23,
						"CONTACT_NAME": "Kolwaczyk",
						"EMAIL":        "rober@email.com",
					},
					},
					PostIncrement: []string{"meta.ACCOUNT_ID", "meta.USER_ID"},
				})

			var state = data.NewMap()
			state.Put("meta", sequences)
			state.Put("tableData", tableData)
			tableRecords, err := endly.AsTableRecords("tableData", state)
			assert.Nil(t, err)
			assert.NotNil(t, tableRecords)
			tableSetupData, ok := tableRecords.(map[string][]map[string]interface{})
			assert.True(t, ok)

			userAccount, hasTable := tableSetupData["USER_ACCOUNT"]
			assert.True(t, hasTable)
			assert.EqualValues(t, 2, len(userAccount))

			prepareRequest := dsunit.NewPrepareRequest(
				dsunit.NewDatasetResource("mydb1", "", "", ""))
			prepareRequest.Data = tableSetupData
			serviceResponse = service.Run(context, prepareRequest)



			assert.Equal(t, "", serviceResponse.Error)
			expectRequest := dsunit.NewExpectRequest(0, dsunit.NewDatasetResource("mydb1", "", "", ""))
			expectRequest.Data = tableSetupData

			serviceResponse = service.Run(context,expectRequest)

			if assert.Equal(t, "", serviceResponse.Error) {
				verifyResponse, ok = serviceResponse.Response.(*dsunit.ExpectResponse)
				if assert.True(t, ok) {
					assert.EqualValues(t, 0, len(verifyResponse.Validation[0].Failures))
				}
			}

		}

	}
	//
	//func TestDsUnitService_Errors(t *testing.T) {
	//	manager := endly.NewManager()
	//	context := manager.NewContext(toolbox.NewContext())
	//	service, err := getRegisteredDsUnitService(manager, context, "mydb1")
	//	if !assert.Nil(t, err) {
	//		return
	//	}
	//	serviceResponse := service.Run(context, &endly.DsUnitRegisterRequest{})
	//	assert.True(t, serviceResponse.Error != "")
	//
	//	serviceResponse = service.Run(context, &endly.DsUnitRegisterRequest{
	//		Datastore: "dd",
	//	})
	//	assert.True(t, serviceResponse.Error != "")
	//
	//	serviceResponse = service.Run(context, &endly.DsUnitRegisterRequest{
	//		Datastore: "dd",
	//		Config: &dsc.Config{
	//			DriverName: "dsads",
	//			Descriptor: "[username]:[password]@tcp(127.0.0.1:3308)/[dbname]?parseTime=true",
	//		},
	//	})
	//
	//	serviceResponse = service.Run(context, &endly.DsUnitPrepareRequest{})
	//	assert.True(t, serviceResponse.Error != "")
	//
	//	serviceResponse = service.Run(context, &endly.DsUnitPrepareRequest{
	//		DsUnitDataRequest: &endly.DsUnitDataRequest{
	//			Datastore: "dd",
	//		}})
	//	assert.True(t, serviceResponse.Error != "")
	//
	//	serviceResponse = service.Run(context, &endly.DsUnitPrepareRequest{
	//		DsUnitDataRequest: &endly.DsUnitDataRequest{
	//			Datastore: "dd",
	//			URL:       url.NewResource("test/nonexisting").URL,
	//		},
	//	})
	//	assert.True(t, serviceResponse.Error != "")
	//
	//	serviceResponse = service.Run(context, &endly.DsUnitSQLRequest{})
	//	assert.True(t, serviceResponse.Error != "")
	//
	//	serviceResponse = service.Run(context, &endly.DsUnitSQLRequest{Datastore: "dd"})
	//	assert.True(t, serviceResponse.Error != "")
	//	serviceResponse = service.Run(context, &endly.DsUnitSQLRequest{
	//		Datastore: "dd",
	//		Scripts: []*url.Resource{
	//			url.NewResource("test/nonexisting"),
	//		},
	//	})
	//	assert.True(t, serviceResponse.Error != "")
	//
	//	serviceResponse = service.Run(context, &endly.DsUnitTableSequenceRequest{})
	//	assert.True(t, serviceResponse.Error != "")
	//
	//	serviceResponse = service.Run(context, &endly.DsUnitTableSequenceRequest{Datastore: "dd"})
	//	assert.True(t, serviceResponse.Error != "")
	//
	//	serviceResponse = service.Run(context, &endly.DsUnitExpectRequest{})
	//	assert.True(t, serviceResponse.Error != "")
	//
	//	serviceResponse = service.Run(context, &endly.DsUnitExpectRequest{DsUnitDataRequest: &endly.DsUnitDataRequest{
	//		Datastore: "dd",
	//	}})
	//	assert.True(t, serviceResponse.Error != "")
	//
	//}
	//



