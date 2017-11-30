package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
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

}
