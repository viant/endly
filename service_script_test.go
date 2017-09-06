package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"io/ioutil"
	"path"
	"testing"
)

func TestScriptService_Run(t *testing.T) {
	manager := endly.NewServiceManager()
	context := manager.NewContext(toolbox.NewContext())
	fileName, _, _ := toolbox.CallerInfo(2)
	parent, _ := path.Split(fileName)
	scriptFile := path.Join(parent, "test/script_test.js")
	content, err := ioutil.ReadFile(scriptFile)
	assert.Nil(t, err)

	scripService, err := context.Service(endly.ScriptServiceId)
	assert.Nil(t, err)
	response := scripService.Run(context, &endly.ScriptCommand{
		Code: string(content),
	})
	assert.Equal(t, "ok", response.Status)
	assert.Nil(t, response.Error)
	var state = context.State()
	assert.True(t, state.Has("127.0.0.1"))

}
