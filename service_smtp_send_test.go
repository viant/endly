package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestSMTPSendRequest_Validate(t *testing.T) {

	{
		var request = &endly.SMTPSendRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &endly.SMTPSendRequest{Target: url.NewResource("abc")}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &endly.SMTPSendRequest{Target: url.NewResource("abc"), Credential: "abc"}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &endly.SMTPSendRequest{Target: url.NewResource("abc"), Credential: "abc", Mail: &endly.MailMessage{}}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &endly.SMTPSendRequest{Target: url.NewResource("abc"), Credential: "abc", Mail: &endly.MailMessage{
			Subject: "abc",
		}}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &endly.SMTPSendRequest{Target: url.NewResource("abc"), Credential: "abc", Mail: &endly.MailMessage{
			Subject: "abc",
			From:    "abc@a.pl",
		}}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &endly.SMTPSendRequest{Target: url.NewResource("abc"), Credential: "abc", Mail: &endly.MailMessage{
			From: "abc@a.pl",
			To:   []string{"abc@a.pl"},
		}}
		assert.NotNil(t, request.Validate())
	}
	{
		var request = &endly.SMTPSendRequest{Target: url.NewResource("abc"), Credential: "abc", Mail: &endly.MailMessage{
			Subject: "abc",
			From:    "abc@a.pl",
			To:      []string{"abc@a.pl"},
		}}
		assert.Nil(t, request.Validate())
	}

}

func TestMailMessage_Receivers(t *testing.T) {
	var mail = &endly.MailMessage{
		Subject: "abc",
		From:    "abc@a.pl",
		To:      []string{"abc@a.pl"},
	}
	assert.EqualValues(t, []string{"abc@a.pl"}, mail.Receivers())
}

func TestMailMessage_Payload(t *testing.T) {

	{
		mail := &endly.MailMessage{
			Subject: "abc",
			From:    "abc@a.pl",
			To:      []string{"abc@a.pl"},
		}
		assert.EqualValues(t, "From: abc@a.pl\r\nTo: abc@a.pl\r\nSubject: abc\r\n\r\n", string(mail.Payload()))

	}

	{
		mail := &endly.MailMessage{
			Subject:     "abc",
			From:        "abc@a.pl",
			To:          []string{"abc@a.pl"},
			ContentType: "text/html",
			Body:        "<h1>Hello</h1>",
		}
		assert.EqualValues(t, "From: abc@a.pl\r\nTo: abc@a.pl\r\nSubject: abc\r\nMIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n<h1>Hello</h1>", string(mail.Payload()))

	}
}
