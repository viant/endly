package core

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNew(t *testing.T) {
	service := New()
	assert.NotNil(t, service)
}


