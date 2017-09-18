package endly_test

import (
	"testing"
	"github.com/viant/endly"
	"github.com/stretchr/testify/assert"
	"strings"
)

func TestNewDefaultState(t *testing.T) {
	state := endly.NewDefaultState()
	var expanded = endly.Expand(state, "home = ${env.HOME} ")
	assert.False(t, strings.Contains( expanded, "${env.HOME}"))
}