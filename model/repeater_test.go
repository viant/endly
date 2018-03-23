package model_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/workflow"
	"github.com/viant/toolbox"
	"testing"
	"github.com/viant/endly/model"
)

func GetAbstractService() *endly.AbstractService {
	manager := endly.New()
	service, _ := manager.Service(workflow.ServiceID)
	nopService := service.(*workflow.Service)
	return nopService.AbstractService
}

func TestRepeatable_Run(t *testing.T) {
	var abstractService = GetAbstractService()

	{ //Test exit criteria with variable extraction from a map

		repeataable := &model.Repeater{
			//Extraction   Extracts //data extraction
			Variables: []*model.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:      10,
			SleepTimeMs: 100,
			Exit:        "$status:!running",
		}

		manager := endly.New()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]interface{})

		var counter = 0
		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			counter++
			if counter < 3 {
				return map[string]interface{}{
					"testStatus": "running",
				}, nil
			}

			return map[string]interface{}{
				"testStatus": "done",
			}, nil
		}, extracted)
		assert.Nil(t, err)
		assert.EqualValues(t, 3, counter)
		assert.EqualValues(t, "done", extracted["status"])
	}

	{ //Test exit criteria with variable extraction from a map

		repeataable := &model.Repeater{
			//Extraction   Extracts //data extraction
			Variables: []*model.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:      10,
			SleepTimeMs: 100,
			Exit:        "$status:!running",
		}

		manager := endly.New()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]interface{})

		var counter = 0
		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			counter++
			if counter < 3 {
				return map[string]interface{}{
					"testStatus": "running",
				}, nil
			}

			return []interface{}{map[string]interface{}{
				"testStatus": "done",
			}, nil}, nil
		}, extracted)
		assert.Nil(t, err)
		assert.EqualValues(t, 3, counter)
		assert.EqualValues(t, "done", extracted["status"])
	}

	{ //Test exit criteria with variable extraction from a JSON text

		repeataable := &model.Repeater{
			//Extraction   Extracts //data extraction
			Variables: []*model.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:      10,
			SleepTimeMs: 100,
			Exit:        "$status:!running",
		}

		manager := endly.New()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]interface{})

		var counter = 0
		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			counter++
			if counter < 3 {
				return `{
					"testStatus": "running"
				}`, nil
			}

			return `{
				"testStatus": "done"
			}`, nil
		}, extracted)
		assert.Nil(t, err)
		assert.EqualValues(t, 3, counter)
		assert.EqualValues(t, "done", extracted["status"])
	}

	{ //Test exit criteria with variable extraction from a []byte  JSON

		repeataable := &model.Repeater{
			//Extraction   Extracts //data extraction
			Variables: []*model.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:      10,
			SleepTimeMs: 100,
			Exit:        "$status:!running",
		}

		manager := endly.New()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]interface{})

		var counter = 0
		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			counter++
			if counter < 3 {
				return []byte(`{
					"testStatus": "running"
				}`), nil
			}

			return []byte(`{
				"testStatus": "done"
			}`), nil
		}, extracted)
		assert.Nil(t, err)
		assert.EqualValues(t, 3, counter)
		assert.EqualValues(t, "done", extracted["status"])
	}

	{ //Test exit criteria with variable extraction from a invalid JSON text vi $value key

		repeataable := &model.Repeater{
			//Extraction   Extracts //data extraction
			Variables: []*model.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:      10,
			SleepTimeMs: 100,
			Exit:        "$value:!/running/", //this is contains
		}

		manager := endly.New()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]interface{})

		var counter = 0
		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			counter++
			if counter < 3 {
				return `{
					"testStatus":"running",
				}`, nil
			}

			return `{
				"testStatus":"done",
			}`, nil
		}, extracted)
		assert.Nil(t, err)
		assert.EqualValues(t, 3, counter)
	}

	{ //Test exit criteria with variable extraction from a invalid JSON text vi $value key

		repeataable := &model.Repeater{
			//Extraction   Extracts //data extraction
			Variables: []*model.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:      10,
			SleepTimeMs: 100,
			Exit:        "$value:!/running/", //this is contains
		}

		manager := endly.New()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]interface{})

		var counter = 0
		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			counter++
			if counter < 3 {
				return `{
					"testStatus":"running",
				}`, nil
			}

			return `{
				"testStatus":"done",
			}`, nil
		}, extracted)
		assert.Nil(t, err)
		assert.EqualValues(t, 3, counter)
	}

	{ //Test data struct to string regexp extraction

		repeataable := &model.Repeater{
			Variables: []*model.Variable{
				{
					Name: "testStatus",
					From: "testStatus",
				},
			},
			//Extraction   Extracts //data extraction
			Extraction: []*model.Extract{
				{
					RegExpr: `"testStatus":"([^"]+)"`,
					Key:     "status",
				},
			},
			Repeat:      10,
			SleepTimeMs: 100,
			Exit:        "$status:!/running/", //this is contains
		}

		manager := endly.New()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]interface{})

		var counter = 0
		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			counter++
			if counter < 3 {
				return map[string]interface{}{
					"testStatus": "running",
				}, nil
			}

			return []interface{}{map[string]interface{}{
				"testStatus": "done",
			}, nil}, nil
		}, extracted)
		assert.Nil(t, err)
		assert.EqualValues(t, 3, counter)
		assert.EqualValues(t, "done", extracted["status"])
		assert.EqualValues(t, "done", extracted["testStatus"])
	}

	{ //Test exit criteria with variable extraction from a invalid JSON text vi $value key

		repeataable := &model.Repeater{
			//Extraction   Extracts //data extraction
			Variables: []*model.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Extraction: []*model.Extract{
				{
					RegExpr: `"testStatus":"([^"]+)"`,
					Key:     "status",
				},
			},
			Repeat:      10,
			SleepTimeMs: 100,
			Exit:        "$value:!/running/", //this is contains
		}

		manager := endly.New()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]interface{})

		var counter = 0
		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			counter++
			if counter < 3 {
				return []interface{}{`{
					"testStatus":"running",
				}`, nil, nil}, nil
			}

			return []interface{}{`{
				"testStatus":"done",
			}`}, nil
		}, extracted)

		assert.Nil(t, err)
		assert.EqualValues(t, 3, counter)
		assert.EqualValues(t, "done", extracted["status"])
	}

	{ //Test  error

		repeataable := &model.Repeater{
			//Extraction   Extracts //data extraction
			Variables: []*model.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:      10,
			SleepTimeMs: 100,
			Exit:        "$status:!running",
		}

		manager := endly.New()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]interface{})

		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			return nil, fmt.Errorf("failed to run test")
		}, extracted)
		assert.NotNil(t, err)

	}

	{ //Test  invalid regexpr

		repeataable := &model.Repeater{
			//Extraction   Extracts //data extraction
			Variables: []*model.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},

			Extraction: []*model.Extract{
				{
					RegExpr: `"testStatus":"(.?+*))"`,
					Key:     "status",
				},
			},
			Repeat:      10,
			SleepTimeMs: 100,
			Exit:        "$status:!running",
		}

		manager := endly.New()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]interface{})

		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			return "abc", nil
		}, extracted)
		assert.NotNil(t, err)

	}

	{ //Test invalid criteria error

		repeataable := &model.Repeater{
			//Extraction   Extracts //data extraction
			Variables: []*model.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},

			Extraction: []*model.Extract{
				{
					RegExpr: `"testStatus":"(.?+*))"`,
					Key:     "status",
				},
			},
			Repeat:      10,
			SleepTimeMs: 100,
			Exit:        "$status!=running",
		}

		manager := endly.New()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]interface{})

		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			return "abc", nil
		}, extracted)
		assert.NotNil(t, err)

	}

}


func TestRepeater_Init(t *testing.T) {

	var repeater *model.Repeater
	repeater = repeater.Init()
	assert.EqualValues(t, 1, repeater.Repeat)


}