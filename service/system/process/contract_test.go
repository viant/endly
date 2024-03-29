package process

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox"
	"path"
	"testing"
)

func Test_NewStartRequestFromURL(t *testing.T) {
	parent := toolbox.CallerDirectory(3)
	req, err := NewStartRequestFromURL(path.Join(parent, "test", "request.json"))
	assert.Nil(t, err)
	assert.NotNil(t, req)
}
