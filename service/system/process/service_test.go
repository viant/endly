package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/endly/service/system/exec"
)

func TestService_buildStartProcessCommand(t *testing.T) {
	testCases := []struct {
		description      string
		request          *StartRequest
		expectedCommands []string
	}{
		{
			description: "regular background process",
			request: &StartRequest{
				Command:   "/tmp/agently-app",
				Arguments: []string{"serve", "--addr", ":9001"},
				Options:   nil,
			},
			expectedCommands: []string{
				"cd  ",
				"touch nohup.out && chmod 666 nohup.out",
				"(/tmp/agently-app serve --addr :9001 &)",
			},
		},
		{
			description: "nohup background process",
			request: &StartRequest{
				Command:   "/tmp/agently-app",
				Arguments: []string{"serve", "--addr", ":9001"},
				Options: &exec.Options{
					Directory: "/tmp/agently",
				},
				ImmuneToHangups: true,
			},
			expectedCommands: []string{
				"cd /tmp/agently ",
				"touch /tmp/agently/nohup.out && chmod 666 /tmp/agently/nohup.out",
				"(nohup /tmp/agently-app serve --addr :9001 &)",
			},
		},
	}

	svc := &service{}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			runRequest := svc.buildStartProcessCommand(testCase.request)
			assert.NotNil(t, runRequest)
			if assert.Len(t, runRequest.Commands, len(testCase.expectedCommands)) {
				for i, expected := range testCase.expectedCommands {
					assert.Equal(t, expected, runRequest.Commands[i].String())
				}
			}
			assert.True(t, runRequest.CheckError)
		})
	}
}
