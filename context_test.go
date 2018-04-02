package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"strings"
	"testing"
	"sync/atomic"
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

func TestContext_Expand_Resource(t *testing.T) {
	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())

	_, err := context.ExpandResource(nil)
	assert.NotNil(t, err)
	_, err = context.ExpandResource(&url.Resource{})
	assert.NotNil(t, err)

	{
		state := context.State()
		state.Put("host", "127.0.0.1")
		expanded, err := context.ExpandResource(url.NewResource("scp://${host}/as"))
		if assert.Nil(t, err) {
			assert.EqualValues(t, "scp://127.0.0.1/as", expanded.URL)
		}
	}
	{
		_, err := context.ExpandResource(url.NewResource("path"))
		assert.Nil(t, err)
	}

}

func TestContext_NewRequest(t *testing.T) {
	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())
	{
		request, err := context.NewRequest("nop", "nop")
		if assert.Nil(t, err) {
			assert.NotNil(t, request)
		}
	}
	{
		_, err := context.NewRequest("invalid", "nop")
		assert.NotNil(t, err)
	}
	{
		_, err := context.NewRequest("nop", "abc")
		assert.NotNil(t, err)
	}

}

func TestContext_AsRequest(t *testing.T) {
	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())
	{
		request, err := context.AsRequest("nop", "nop", map[string]interface{}{})
		if assert.Nil(t, err) {
			assert.NotNil(t, request)
		}
	}
	{
		_, err := context.AsRequest("zzz", "nop", map[string]interface{}{})
		assert.NotNil(t, err)
	}

}

func TestContext_Clone(t *testing.T) {
	manager := endly.New()
	{
		context := manager.NewContext(toolbox.NewContext())
		cloned := context.Clone()
		assert.NotNil(t, cloned)

	}
	{
		var deferredRun int32 = 0
		context := manager.NewContext(toolbox.NewContext())
		context.Deffer(func() {
			atomic.AddInt32(&deferredRun, 1)
		})
		cloned := context.Clone()
		assert.False(t, cloned.IsClosed())
		context.Close()
		assert.True(t, cloned.IsClosed())
		assert.EqualValues(t, int32(2), deferredRun)
	}

}
