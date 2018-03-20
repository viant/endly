package smtp_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"

	"github.com/viant/endly/notify/smtp"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"path"
	"testing"
	"time"
)

func TestService_Run(t *testing.T) {
	var parent = toolbox.CallerDirectory(3)
	credentials := path.Join(parent, "test/secret.json")
	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())
	service, _ := context.Service(smtp.ServiceID)
	{ //missing subject
		serviceResponse := service.Run(context, &smtp.SendRequest{
			Target: url.NewResource("smtp://smtp.gmail.com:465", credentials+"a"),
			Mail: &smtp.MailMessage{
				From:        "adrianwit@gmail.com",
				To:          []string{"viantemailtester@gmail.com"},
				ContentType: "text/html",
				Body:        "<h1>Hello</h1>",
			},
		})
		assert.True(t, serviceResponse.Error != "")
	}
	{ //invalid credentials subject
		serviceResponse := service.Run(context, &smtp.SendRequest{
			Target: url.NewResource("smtp://smtp.gmail.com:465", credentials+"aa"),
			Mail: &smtp.MailMessage{
				From:        "adrianwit@gmail.com",
				To:          []string{"viantemailtester@gmail.com"},
				ContentType: "text/html",
				Body:        "<h1>Hello</h1>",
			},
		})
		assert.True(t, serviceResponse.Error != "")
	}
	{ //sending message
		serviceResponse := service.Run(context, &smtp.SendRequest{
			Target: url.NewResource("smtp://smtp.gmail.com:465", credentials),
			Mail: &smtp.MailMessage{
				From:        "adrianwit@gmail.com",
				To:          []string{"viantemailtester@gmail.com"},
				ContentType: "text/html",
				Subject:     fmt.Sprintf("Sedning HTML grettings at  %v-%v", time.Now().Month(), time.Now().Day()),
				Body:        "<h1>Hello</h1>",
			},
		})
		assert.EqualValues(t, "", serviceResponse.Error)
		response, ok := serviceResponse.Response.(*smtp.SendResponse)
		if assert.True(t, ok) {
			if response != nil {
				assert.True(t, response.SendPayloadSize > 0)
			}

		}
	}
}

func TestNewSMTPClient(t *testing.T) {

	{

		var target = url.NewResource("smtp://smtp.gmail.com:465")
		var parent = toolbox.CallerDirectory(3)
		credentials := path.Join(parent, "test/secret.json")
		client, err := smtp.NewClient(target, credentials)
		if assert.Nil(t, err) {
			assert.NotNil(t, client)
			client.Close()
		}
	}

	{

		var target = url.NewResource("smtp://smtp.gmail.com:465")
		var parent = toolbox.CallerDirectory(3)
		credentials := path.Join(parent, "test/invalid_secret.json")
		_, err := smtp.NewClient(target, credentials)
		assert.NotNil(t, err)
	}

}
