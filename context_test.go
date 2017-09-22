package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"strings"
	"testing"
)

func TestNewDefaultState(t *testing.T) {
	state := endly.NewDefaultState()
	var expanded = endly.Expand(state, "home = ${env.HOME} ")
	assert.False(t, strings.Contains(expanded, "${env.HOME}"))
}
