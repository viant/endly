package shared

import (
	"github.com/go-errors/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsNotFound(t *testing.T) {

	{
		notFound := &NotFound{Message: "not found"}
		assert.True(t, IsNotFound(notFound))
	}
	{
		notFound := errors.New("not found")
		assert.True(t, IsNotFound(notFound))
	}
	{
		notFound := errors.New("not")
		assert.False(t, IsNotFound(notFound))
	}
	{
		notFound := &NotFound{Message: "abc"}
		assert.True(t, IsNotFound(notFound))
	}
}
