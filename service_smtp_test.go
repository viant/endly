package endly_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"path"
	"testing"
	"time"
)

func TestSmtpService_Run(t *testing.T) {
	var parent = toolbox.CallerDirectory(3)
	credential := path.Join(parent, "test/smtp/secret.json")
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	service, _ := context.Service(endly.SMTPServiceID)
	{ //missing subject
		serviceResponse := service.Run(context, &endly.SMTPSendRequest{
			Credential: credential,
			Target:     url.NewResource("smtp://smtp.gmail.com:465"),
			Mail: &endly.MailMessage{
				From:        "adrianwit@gmail.com",
				To:          []string{"viantemailtester@gmail.com"},
				ContentType: "text/html",
				Body:        "<h1>Hello</h1>",
			},
		})
		assert.True(t, serviceResponse.Error != "")
	}
	{ //invalid credentials subject
		serviceResponse := service.Run(context, &endly.SMTPSendRequest{
			Credential: credential + "aa",
			Target:     url.NewResource("smtp://smtp.gmail.com:465"),
			Mail: &endly.MailMessage{
				From:        "adrianwit@gmail.com",
				To:          []string{"viantemailtester@gmail.com"},
				ContentType: "text/html",
				Body:        "<h1>Hello</h1>",
			},
		})
		assert.True(t, serviceResponse.Error != "")
	}
	{ //sending message
		serviceResponse := service.Run(context, &endly.SMTPSendRequest{
			Credential: credential,
			Target:     url.NewResource("smtp://smtp.gmail.com:465"),
			Mail: &endly.MailMessage{
				From:        "adrianwit@gmail.com",
				To:          []string{"viantemailtester@gmail.com"},
				ContentType: "text/html",
				Subject:     fmt.Sprintf("Sedning HTML grettings at  %v-%v", time.Now().Month(), time.Now().Day()),
				Body:        "<h1>Hello</h1>",
			},
		})
		assert.EqualValues(t, "", serviceResponse.Error)
		response, ok := serviceResponse.Response.(*endly.SMTPSendResponse)
		if assert.True(t, ok) {
			assert.EqualValues(t, 185, response.SendPayloadSize)

		}
	}
}

func TestNewSMTPClient(t *testing.T) {

	{

		var target = url.NewResource("smtp://smtp.gmail.com:465")
		var parent = toolbox.CallerDirectory(3)
		credential := path.Join(parent, "test/smtp/secret.json")
		client, err := endly.NewSMTPClient(target, credential)
		if assert.Nil(t, err) {
			assert.NotNil(t, client)
			client.Close()
		}
	}

	{

		var target = url.NewResource("smtp://smtp.gmail.com:465")
		var parent = toolbox.CallerDirectory(3)
		credential := path.Join(parent, "test/smtp/invalid_secret.json")
		_, err := endly.NewSMTPClient(target, credential)
		assert.NotNil(t, err)
	}

}
