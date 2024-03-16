package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/endly/model/location"
	"github.com/viant/toolbox"
	"gopkg.in/yaml.v2"
	"path"
	"strings"
	"testing"
)

func TestTemplate_Expand(t *testing.T) {

	parentDir := toolbox.CallerDirectory(3)

	var useCases = []struct {
		description       string
		workflowPrefixURL string
		hasError          bool
	}{
		{
			description:       "simple template",
			workflowPrefixURL: path.Join(parentDir, "test/inline", "simple_template"),
		},
		{
			description:       "group template",
			workflowPrefixURL: path.Join(parentDir, "test/inline", "group_template"),
		},
		{
			description:       "simple subpath template",
			workflowPrefixURL: path.Join(parentDir, "test/inline", "simple_subpath"),
		},
		{
			description:       "data template",
			workflowPrefixURL: path.Join(parentDir, "test/inline", "data"),
		},
	}

	for _, useCase := range useCases {
		var inputURL = useCase.workflowPrefixURL + ".yaml"
		baseURL, name := toolbox.URLSplit(inputURL)

		name = string(name[:len(name)-len(path.Ext(name))])

		inlineWorkflow, err := loadInlineWorkflow(inputURL)

		if useCase.hasError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		if !assert.Nil(t, err, useCase.description) {
			continue
		}

		if !assert.NotNil(t, inlineWorkflow, useCase.description) {
			continue
		}

		workflow, err := inlineWorkflow.AsWorkflow(name, baseURL)
		if !assert.Nil(t, err, useCase.description) {
			continue
		}

		var expectedURL = useCase.workflowPrefixURL + ".json"
		expectedResource := location.NewResource(expectedURL)
		if expected, err := expectedResource.DownloadText(); err == nil {
			if !assertly.AssertValues(t, expected, workflow, useCase.description) {
				toolbox.DumpIndent(workflow, true)
			}
		}
	}
	//TODO error case (without @)
	// with invalida data json

}

func loadInlineWorkflow(URL string) (*InlineWorkflow, error) {
	var inlineURL = location.NewResource(URL)
	YAMLText, err := inlineURL.DownloadText()
	if err != nil {
		return nil, err
	}
	var YAML = &yaml.MapSlice{}
	if err = yaml.NewDecoder(strings.NewReader(YAMLText)).Decode(YAML); err != nil {
		return nil, err
	}

	aMap := map[string]interface{}{}
	for _, entry := range *YAML {
		aMap[toolbox.AsString(entry.Key)] = entry.Value
	}
	inline := &InlineWorkflow{}
	err = toolbox.DefaultConverter.AssignConverted(inline, aMap)
	return inline, err
}
