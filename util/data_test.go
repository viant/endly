package util

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"path"
	"testing"
)

func TestLoadMap(t *testing.T) {

	useCases := []struct {
		description string
		URI         string
		expect      interface{}
		hasError    bool
	}{
		{
			description: "simple map",
			URI:         "@simple_map",
			expect: `{
	"k1": 1,
	"k2": "abc"
}`,
		},

		{
			description: "expanded_map",
			URI:         "@expanded_map @submap",
			expect: `{
	"k1": 1,
	"k2": {
		"k11": 1,
		"k12": 2
	}
}`,
		},

		{
			description: "expanded_map with args",
			URI:         "@map_inline @map_arg | @map_arg",
			expect: `{
	"k1": 1,
	"k2": {
		"a": [1,2]
	},
	"k3": {
		"a": [1,2]
	}
}
`,
		},
	}
	//map_inline

	parentDirectory := path.Join(toolbox.CallerDirectory(3), "test")

	for _, useCase := range useCases {
		aMap, err := LoadMap([]string{url.NewResource(parentDirectory).URL}, useCase.URI)
		if useCase.hasError {
			assert.NotNil(t, err, useCase.description)
			continue
		}

		if !assert.Nil(t, err, useCase.description) {
			continue
		}
		assertly.AssertValues(t, useCase.expect, aMap, useCase.description)
	}

}

func TestListResource(t *testing.T) {
	parentDirectory := path.Join(toolbox.CallerDirectory(3), "test")
	URLs, err := ListResource([]string{parentDirectory}, "dir/*_xx")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(URLs))

}
