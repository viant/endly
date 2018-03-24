package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"strings"
	"testing"
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

}
