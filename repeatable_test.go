package endly_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
)

func TestRepeatable_Run(t *testing.T) {
	var abstractService = GetAbstractService()

	{ //Test exit criteria with variable extraction from a map

		repeataable := &endly.Repeatable{
			//Extraction   DataExtractions //data extraction
			Variables: []*endly.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:       10,
			SleepTimeMs:  100,
			ExitCriteria: "$status:!running",
		}

		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]string)

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

		repeataable := &endly.Repeatable{
			//Extraction   DataExtractions //data extraction
			Variables: []*endly.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:       10,
			SleepTimeMs:  100,
			ExitCriteria: "$status:!running",
		}

		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]string)

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

		repeataable := &endly.Repeatable{
			//Extraction   DataExtractions //data extraction
			Variables: []*endly.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:       10,
			SleepTimeMs:  100,
			ExitCriteria: "$status:!running",
		}

		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]string)

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

		repeataable := &endly.Repeatable{
			//Extraction   DataExtractions //data extraction
			Variables: []*endly.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:       10,
			SleepTimeMs:  100,
			ExitCriteria: "$status:!running",
		}

		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]string)

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

		repeataable := &endly.Repeatable{
			//Extraction   DataExtractions //data extraction
			Variables: []*endly.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:       10,
			SleepTimeMs:  100,
			ExitCriteria: "$value:!/running/", //this is contains
		}

		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]string)

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

		repeataable := &endly.Repeatable{
			//Extraction   DataExtractions //data extraction
			Variables: []*endly.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:       10,
			SleepTimeMs:  100,
			ExitCriteria: "$value:!/running/", //this is contains
		}

		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]string)

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

		repeataable := &endly.Repeatable{
			Variables: []*endly.Variable{
				{
					Name: "testStatus",
					From: "testStatus",
				},
			},
			//Extraction   DataExtractions //data extraction
			Extraction: []*endly.DataExtraction{
				{
					RegExpr: `"testStatus":"([^"]+)"`,
					Key:     "status",
				},
			},
			Repeat:       10,
			SleepTimeMs:  100,
			ExitCriteria: "$status:!/running/", //this is contains
		}

		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]string)

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

		repeataable := &endly.Repeatable{
			//Extraction   DataExtractions //data extraction
			Variables: []*endly.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Extraction: []*endly.DataExtraction{
				{
					RegExpr: `"testStatus":"([^"]+)"`,
					Key:     "status",
				},
			},
			Repeat:       10,
			SleepTimeMs:  100,
			ExitCriteria: "$value:!/running/", //this is contains
		}

		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]string)

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

		repeataable := &endly.Repeatable{
			//Extraction   DataExtractions //data extraction
			Variables: []*endly.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},
			Repeat:       10,
			SleepTimeMs:  100,
			ExitCriteria: "$status:!running",
		}

		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]string)

		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			return nil, fmt.Errorf("failed to run test")
		}, extracted)
		assert.NotNil(t, err)

	}

	{ //Test  invalid regexpr

		repeataable := &endly.Repeatable{
			//Extraction   DataExtractions //data extraction
			Variables: []*endly.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},

			Extraction: []*endly.DataExtraction{
				{
					RegExpr: `"testStatus":"(.?+*))"`,
					Key:     "status",
				},
			},
			Repeat:       10,
			SleepTimeMs:  100,
			ExitCriteria: "$status:!running",
		}

		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]string)

		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			return "abc", nil
		}, extracted)
		assert.NotNil(t, err)

	}

	{ //Test invalid criteria error

		repeataable := &endly.Repeatable{
			//Extraction   DataExtractions //data extraction
			Variables: []*endly.Variable{
				{
					Name: "status",
					From: "testStatus",
				},
			},

			Extraction: []*endly.DataExtraction{
				{
					RegExpr: `"testStatus":"(.?+*))"`,
					Key:     "status",
				},
			},
			Repeat:       10,
			SleepTimeMs:  100,
			ExitCriteria: "$status!=running",
		}

		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		var extracted = make(map[string]string)

		err := repeataable.Run(abstractService, "test1", context, func() (interface{}, error) {
			return "abc", nil
		}, extracted)
		assert.NotNil(t, err)

	}

}
