package endly

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractPathIndex(t *testing.T) {
	var index, path = ExtractPathIndex("[/[1][Code]")
	assert.Equal(t, 1, index)
	assert.Equal(t, "[Code]", path)
}
