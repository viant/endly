package smtp_test

import (
	"github.com/stretchr/testify/assert"

	"github.com/viant/endly/notify/smtp"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestSMTPSendRequest_Validate(t *testing.T) {

	{
		var request = &smtp.SendRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &smtp.SendRequest{Target: url.NewResource("abc")}
		assert.NotNil(t, request.Validate())
	}
	{

		var request = &smtp.SendRequest{Target: url.NewResource("abc", "abc")}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &smtp.SendRequest{Target: url.NewResource("abc", "abc"), Mail: &smtp.Message{}}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &smtp.SendRequest{Target: url.NewResource("abc", "abc"), Mail: &smtp.Message{
			Subject: "abc",
		}}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &smtp.SendRequest{Target: url.NewResource("abc", "abc"), Mail: &smtp.Message{
			Subject: "abc",
			From:    "abc@a.pl",
		}}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &smtp.SendRequest{Target: url.NewResource("abc", "abc"), Mail: &smtp.Message{
			From: "abc@a.pl",
			To:   []string{"abc@a.pl"},
		}}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &smtp.SendRequest{Target: url.NewResource("abc", "abc"), Mail: &smtp.Message{
			Subject: "abc",
			From:    "abc@a.pl",
			To:      []string{"abc@a.pl"},
		}}
		assert.Nil(t, request.Validate())
	}

}

func TestMailMessage_Receivers(t *testing.T) {
	var mail = &smtp.Message{
		Subject: "abc",
		From:    "abc@a.pl",
		To:      []string{"abc@a.pl"},
	}
	assert.EqualValues(t, []string{"abc@a.pl"}, mail.Receivers())
}

func TestMailMessage_Payload(t *testing.T) {

	{
		mail := &smtp.Message{
			Subject: "abc",
			From:    "abc@a.pl",
			To:      []string{"abc@a.pl"},
		}
		assert.EqualValues(t, "From: abc@a.pl\r\nTo: abc@a.pl\r\nSubject: abc\r\n\r\n", string(mail.Payload()))

	}

	{
		mail := &smtp.Message{
			Subject:     "abc",
			From:        "abc@a.pl",
			To:          []string{"abc@a.pl"},
			ContentType: "text/html",
			Body:        "<h1>Hello</h1>",
		}
		assert.EqualValues(t, "From: abc@a.pl\r\nTo: abc@a.pl\r\nSubject: abc\r\nMIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n<h1>Hello</h1>", string(mail.Payload()))

	}
}
