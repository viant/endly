package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
)

func TestExtracts_Extract(t *testing.T) {

	ctx := endly.New().NewContext(nil)
	useCases := []struct {
		desription       string
		extracts         Extracts
		inputs           []string
		expected         map[string]interface{}
		hasError         bool
		alreadyExtracted map[string]interface{}
	}{
		{
			desription: "single line expresssion",
			extracts: []*Extract{
				{
					Key:     "status",
					RegExpr: `"testStatus":"([^\"]+)"`,
				},
			},
			inputs: []string{
				`"testStatus":"running"`,
			},
			expected: map[string]interface{}{
				"status": "running",
			},
		},
		{
			desription: "multi line expr",
			extracts: []*Extract{
				{
					Key:     "url",
					RegExpr: `(?sm).+httpsTrigger:[^u]+url:[\s\t]+([^\n]+)`,
				},
			},
			inputs: strings.Split(`Deploying function (may take a while - up to 2 minutes)...done.                                    
availableMemoryMb: 256
entryPoint: MyFun
httpsTrigger:
  url: https://us-central1-myproj.cloudfunctions.net/MyFun
labels:
  deployment-tool: cli-gcloud
name: projects/us-central1-myproj.cloudfunctions.net/MyFun
runtime: go111
status: ACTIVE
timeout: 60s
updateTime: '2019-01-03T19:24:18Z'
versionId: '2'`, "\n"),
			expected: map[string]interface{}{
				"url": "https://us-central1-myproj.cloudfunctions.net/MyFun",
			},
		},
		{
			desription: "multi single line expr",
			extracts: []*Extract{
				{
					Key:     "status",
					RegExpr: `"testStatus":"([^\"]+)"`,
				},
			},
			inputs: []string{
				`"runtatus":"running"`,
				`"testStatus":"running"`,
			},
			expected: map[string]interface{}{
				"status": "running",
			},
		},

		{
			desription: "invalid expr",
			extracts: []*Extract{
				{
					Key:     "status",
					RegExpr: `"testStatus":"(.?+*))"`,
				},
			},
			inputs: []string{
				`"runtatus":"running"`,
				`"testStatus":"running"`,
			},
			hasError: true,
		},
		{
			desription: "single line required expression",
			extracts: []*Extract{
				{
					Key:      "status",
					RegExpr:  `"testStatus":"([^\"]+)"`,
					Required: true,
				},
			},
			inputs: []string{
				`"testStatus":"running"`,
			},
			expected: map[string]interface{}{
				"status": "running",
			},
		},
		{
			desription: "single line no capture group",
			extracts: []*Extract{
				{
					Key:      "status",
					RegExpr:  `"testStatus":"[^\"]+"`,
					Required: true,
				},
			},
			inputs: []string{
				`"testStatus":"running"`,
			},
			expected: map[string]interface{}{
				"status": `"testStatus":"running"`,
			},
		},
		{
			desription: "single line no capture group, no match",
			extracts: []*Extract{
				{
					Key:      "status",
					RegExpr:  `"test-status":"[^\"]+"`,
					Required: true,
				},
			},
			inputs: []string{
				`"testStatus":"running"`,
			},
			hasError: true,
		},
		{
			desription: "single line missing required expression",
			extracts: []*Extract{
				{
					Key:      "status",
					RegExpr:  `"testStatus":"([^\"]+)"`,
					Required: true,
				},
			},
			inputs: []string{
				`"runtatus":"running"`,
			},
			expected: map[string]interface{}{},
			hasError: true,
		},
		{
			desription: "override existing variable",
			extracts: []*Extract{
				{
					Key:      "status",
					RegExpr:  `"testStatus":"([^\"]+)"`,
					Required: true,
				},
			},
			inputs: []string{
				`"testStatus":"stopped"`,
			},
			expected: map[string]interface{}{
				"status": "stopped",
			},
			alreadyExtracted: map[string]interface{}{
				"status": "running",
			},
		},
		{
			desription: "no error when no match, but var already exists",
			extracts: []*Extract{
				{
					Key:      "status",
					RegExpr:  `"testStatus":"([^\"]+)"`,
					Required: true,
				},
			},
			inputs: []string{
				`"status":"stopped"`,
			},
			expected: map[string]interface{}{
				"status": "running",
			},
			alreadyExtracted: map[string]interface{}{
				"status": "running",
			},
		},
	}

	for _, useCase := range useCases {

		var actual = map[string]interface{}{}
		if useCase.alreadyExtracted != nil {
			actual = useCase.alreadyExtracted
		}
		err := useCase.extracts.Extract(ctx, actual, useCase.inputs...)
		if useCase.hasError {
			assert.NotNil(t, err, useCase.desription)
			continue
		}
		if !assert.Nil(t, err, useCase.desription) {
			continue
		}
		assert.EqualValues(t, useCase.expected, actual, useCase.desription)
	}

}
