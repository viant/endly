package selenium

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/toolbox/url"
)



func TestNewRunRequestFromURL(t *testing.T) {

	var useCases = []struct {
		Description string
		URL         string
		Expected    interface{}
		HasError    bool
	}{
		{
			Description: "Commands test",
			URL:         "test/run1.yaml",
			Expected: `{
  "Actions": [
    {
      "Calls": [
        {
          "Method": "Get",
          "Parameters": [
            "http://play.golang.org/?simple=1"
          ]
        }
      ]
    },
    {
      "Selector": {
        "By": "css selector",
        "Value": "#code"
      },
      "Calls": [
        {
          "Method": "Clear"
        },
        {
          "Method": "SendKeys",
          "Parameters": [
            "$Cat(test/code.go)"
          ]
        }
      ]
    },
    {
      "Selector": {
        "By": "css selector",
        "Value": "#run"
      },
      "Calls": [
        {
          "Method": "Click"
        }
      ]
    },
    {
      "Key": "output",
      "Selector": {
        "By": "css selector",
        "Value": "#output",
        "Key": "output"
      },
      "Calls": [
        {
          "Method": "Text"
        }
      ]
    },
    {
      "Calls": [
        {
          "Method": "Close"
        }
      ]
    }
  ]
}`,
		},
		{
			Description: "Commands test with wait",
			URL:         "test/run2.yaml",
			Expected: `{
  "Actions": [
    {
      "Calls": [
        {
          "Method": "Get",
          "Parameters": [
            "http://play.golang.org/?simple=1"
          ]
        }
      ]
    },
    {
      "Selector": {
        "By": "css selector",
        "Value": "#code"
      },
      "Calls": [
        {
          "Method": "Clear"
        },
        {
          "Method": "SendKeys",
          "Parameters": [
            "$Cat(test/code.go)"
          ]
        }
      ]
    },
    {
      "Selector": {
        "By": "css selector",
        "Value": "#run"
      },
      "Calls": [
        {
          "Method": "Click"
        }
      ]
    },
    {
      "Key": "output",
      "Selector": {
        "By": "css selector",
        "Value": "#output",
        "Key": "output"
      },
      "Calls": [
        {
          "Wait": {
            "Repeat": 10,
            "SleepTimeMs": 1000,
            "Exit": "$output:/Endly/"
          },
          "Method": "Text"
        }
      ]
    },
    {
      "Calls": [
        {
          "Method": "Close"
        }
      ]
    }
  ]
}`,},
		{
			Description: "Commands test with wait missing command error",
			URL:         "test/run3.yaml",
			HasError:    true,
		},
		{
			Description: "Commands test with wait error",
			URL:         "test/run4.yaml",
			HasError:    true,
		},
		{
			Description: "Commands yaml syntax error",
			URL:         "test/run5.yaml",
			HasError:    true,
		},
		{
			Description: "Commands parsing error",
			URL:         "test/run6.yaml",
			HasError:    true,
		},
	}

	for _, useCase := range useCases {
		request, err := NewRunRequestFromURL(useCase.URL)
		if err == nil {
			err = request.Init()
		}
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		if !assert.Nil(t, err, useCase.Description) {
			continue
		}
		assertly.AssertValues(t, useCase.Expected, request)

	}

}


func TestNewAction(t *testing.T) {
	var action = NewAction("k1", "#name", "clear")
	assert.EqualValues(t, "clear", action.Calls[0].Method)
	assert.EqualValues(t, "#name", action.Selector.Value)
	assert.EqualValues(t, "k1", action.Selector.Key)

}


func TestRunRequest_Validate(t *testing.T) {

	{
		var req =  NewRunRequest("", "", nil, nil)
		assert.NotNil(t, req.Validate(), "empty remote")
	}
	{
		var req =  NewRunRequest("", "", url.NewResource("abc"), nil)
		assert.NotNil(t, req.Validate(), "empty browser")
	}

	{
		var req =  NewRunRequest("", "firefox", url.NewResource("abc"))
		assert.NotNil(t, req.Validate(), "action empty")
	}
	{
		var req =  NewRunRequest("", "firefox", url.NewResource("abc"), NewAction("", "", "get", "localhost"))
		assert.Nil(t, req.Validate(), "valid request")
	}
	{
		var req =  NewRunRequest("123", "", nil,  NewAction("", "", "get", "localhost"))
		assert.Nil(t, req.Validate(), "valid request")
	}
}


func TestRunRequest_Init(t *testing.T) {

	{//init action selector
		var req = NewRunRequest("123", "", nil,
			NewAction("", "", "get", "localhost"),
			NewAction("", "#name", "clear", "localhost"))
		assert.Nil(t, req.Init())
		assert.EqualValues(t, "css selector", req.Actions[1].Selector.By)
	}

	{
		var req = NewRunRequest("123", "", nil)
		assert.Nil(t, req.Init())
	}
}

func TestOpenSessionRequest_Validate(t *testing.T) {
	{
		var req= NewOpenSessionRequest("forefox", nil)
		assert.NotNil(t, req.Validate())
	}
	{
		var req= NewOpenSessionRequest("forefox", &url.Resource{})
		assert.NotNil(t, req.Validate())
	}
	{
		var req= NewOpenSessionRequest("", url.NewResource("abc"))
		assert.NotNil(t, req.Validate())
	}

	{
		var req= NewOpenSessionRequest("forefox", url.NewResource("abc"))
		assert.Nil(t, req.Validate())
	}
}

func TestNewMethodCall(t *testing.T) {
	var m = NewMethodCall("get", nil, nil)
	assert.EqualValues(t, "get" ,m.Method)
}


func TestNewStartRequestFromURL(t *testing.T) {
	req , err := NewStartRequestFromURL("test/start.yaml")
	assert.Nil(t, err)
	assert.EqualValues(t, "jdk", req.Sdk)
	assert.EqualValues(t, "1.8", req.SdkVersion)
	assert.EqualValues(t, 8085, req.Port)
	assert.EqualValues(t, "3.4.0", req.Version)
	assert.EqualValues(t, "ssh://127.0.0.1/", req.Target.URL)

}

func TestNewStopRequestFromURL(t *testing.T) {
	req , err := NewStopRequestFromURL("test/start.yaml")
	assert.Nil(t, err)
	assert.EqualValues(t, 8085, req.Port)
	assert.EqualValues(t, "ssh://127.0.0.1/", req.Target.URL)

}



func TestNewCloseRequestFromURL(t *testing.T) {
	req , err := NewCloseSessionRequestFromURL("test/close.yaml")
	assert.Nil(t, err)
	assert.EqualValues(t, "abc", req.SessionID)
}
