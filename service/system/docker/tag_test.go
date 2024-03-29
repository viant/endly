package docker

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTag(t *testing.T) {

	var useCases = []struct {
		description string
		tag         string
	}{

		{
			description: "basic tag",
			tag:         "mysql:3.6",
		},
		{
			description: "basic tag with repo",
			tag:         "endly/endly:0.32.2-alpine3.8",
		},
		{
			description: "gcr image",
			tag:         " gcr.io/google-samples/node-hello",
		},
	}

	for _, useCase := range useCases {
		tag := NewTag(useCase.tag)
		assert.Equal(t, tag.String(), useCase.tag, useCase.description)
	}

}
