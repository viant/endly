package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"strings"
	"testing"
	"github.com/viant/toolbox"
	"fmt"
	"github.com/viant/toolbox/url"
	"github.com/viant/toolbox/storage"
)

func TestNewDefaultState(t *testing.T) {
	state := endly.NewDefaultState()

	for _, expr := range []string{"$rand", "${env.HOME}", "$time", "$ts", "$tmpDir", "$uuid.get", "$uuid.next", "$timestamp.now", "$timestamp.tomorrow", "$timestamp.yesterday"} {
		var expanded = state.ExpandAsText(expr)
		assert.False(t, strings.Contains(expanded, expr))
		assert.True(t, len(expr) > 0)
	}

	var expanded = state.ExpandAsText("${tmpDir.subdir}")
	assert.Contains(t, expanded, "/subdir")

}

func TestContext_AsRequest(t *testing.T) {

	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())

	nopRequest, err :=context.AsRequest("nop", "nop", map[string]interface{}{})
	assert.Nil(t, err)
	assert.EqualValues(t, fmt.Sprintf("%T", nopRequest), fmt.Sprintf("%T", &endly.Nop{}))


	_, err =context.AsRequest("abc", "nop", map[string]interface{}{})
	assert.NotNil(t, err)
	_, err =context.AsRequest("nop", "abc", map[string]interface{}{})
	assert.NotNil(t, err)

}

func TestContext_Expand_Resource(t *testing.T) {
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())

	_, err := context.ExpandResource(nil)
	assert.NotNil(t, err)
	_, err = context.ExpandResource(&url.Resource{})
	assert.NotNil(t, err)

}

func TestContext_TerminalSession(t *testing.T) {
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	_, err := context.TerminalSession(url.NewResource("mem:///"))
	assert.NotNil(t, err)
}


func TestContext_Execute_WithError(t *testing.T) {
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	_, err := context.Execute(url.NewResource("mem:///"), []string{"cd /etc","ls -al"})
	assert.NotNil(t, err)
}

func TestContext_Copy(t *testing.T) {
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	memService := storage.NewMemoryService()
	memService.Upload("mem:///a.txt", strings.NewReader("abc"))
	_, err := context.Copy(true, url.NewResource("mem:///a.txt"), url.NewResource("mem:///b.txt"))
	assert.Nil(t, err)
}

func TestContext_Transfer(t *testing.T) {
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	memService := storage.NewMemoryService()
	memService.Upload("mem:///a.txt", strings.NewReader("abc"))
	_, err := context.Transfer(&endly.Transfer{
		Source:url.NewResource("mem:///aaa.txt"),
		Target:url.NewResource("mem:///bbb.txt"),
	})
	assert.NotNil(t, err)
}