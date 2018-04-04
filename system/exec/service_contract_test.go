package exec

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestNewExtractRequest(t *testing.T) {
	{
		request := NewExtractRequest(url.NewResource("scp://127.0.0.1"), DefaultOptions(), NewExtractCommand("ls -al", "", nil, nil))
		assert.NotNil(t, request)
		assert.EqualValues(t, "ls -al", request.Commands[0].Command)
		assert.EqualValues(t, "scp://127.0.0.1", request.Target.URL)
	}

}

func TestRunRequest_Init(t *testing.T) {
	{
		request := NewExtractRequest(url.NewResource("scp://127.0.0.1"), DefaultOptions())
		assert.NotNil(t, request.Validate())
	}
	{
		request := NewExtractRequest(nil, DefaultOptions(), NewExtractCommand("ls -al", "", nil, nil))
		assert.NotNil(t, request.Validate())
	}
	{
		request := NewExtractRequest(url.NewResource("scp://127.0.0.1"), DefaultOptions(), NewExtractCommand("ls -al", "", nil, nil))
		assert.Nil(t, request.Validate())

	}
}

func TestRunRequest_AsExtractRequest(t *testing.T) {
	request := NewRunRequest(url.NewResource("scp://127.0.0.1"), false, "whoami", "$stdout:/awitas/ ? echo 'hello'")
	extract := request.AsExtractRequest()
	if assert.NotNil(t, extract) {
		assert.NotNil(t, request)
		assert.EqualValues(t, "whoami", extract.Commands[0].Command)
		assert.EqualValues(t, "echo 'hello'", extract.Commands[1].Command)
		assert.EqualValues(t, "$stdout:/awitas/ ", extract.Commands[1].When)
		assert.EqualValues(t, "scp://127.0.0.1", extract.Target.URL)
	}
}

func TestNewExtractRequestFromURL(t *testing.T) {
	{
		request, err := NewExtractRequestFromURL("test/extract/req1.json")
		if assert.Nil(t, err) {
			assert.NotNil(t, request)
			assert.EqualValues(t, "ls -al", request.Commands[0].Command)
			assert.EqualValues(t, "ls -al /home/", request.Commands[1].Command)
			assert.EqualValues(t, "$stdout:/home/", request.Commands[1].When)
			assert.EqualValues(t, "scp://127.0.0.1", request.Target.URL)
			assert.EqualValues(t, 3200, request.TimeoutMs)
		}
	}
}

func TestCommand_WhenAndCommand(t *testing.T) {

	var useCases = []struct {
		Description     string
		Expression      string
		ExpectedWhen    string
		ExpectedCommand string
	}{
		{
			Description:     "simple when and command ",
			Expression:      "$stdout:/end/? q",
			ExpectedWhen:    "$stdout:/end/",
			ExpectedCommand: "q",
		},
		{
			Description:     "when and command ",
			Expression:      "$stdout:/end/ AND $cound >  1 ? q",
			ExpectedWhen:    "$stdout:/end/ AND $cound >  1 ",
			ExpectedCommand: "q",
		},
		{
			Description:     "command with partial when (thread as command)",
			Expression:      "echo ':/end/  ? q'",
			ExpectedWhen:    "",
			ExpectedCommand: "echo ':/end/  ? q'",
		},
		{
			Description:     "command",
			Expression:      "ls -al",
			ExpectedWhen:    "",
			ExpectedCommand: "ls -al",
		},
	}

	for _, useCase := range useCases {
		var expr = Command(useCase.Expression)
		when, command := expr.WhenAndCommand()
		assert.EqualValues(t, useCase.ExpectedWhen, when, useCase.Description)
		assert.EqualValues(t, useCase.ExpectedCommand, command, useCase.Description)

	}

}
