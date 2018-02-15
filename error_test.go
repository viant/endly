package endly

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestError_Error(t *testing.T) {
	var e = NewError("s1", "a1", fmt.Errorf("test error 1"))
	assert.Equal(t, "test error 1 at s1.a1", e.Error())
	var e1 = NewError("s2", "a2", e)
	assert.Equal(t, "test error 1 at s2.a2/s1.a1", e1.Error())
}
