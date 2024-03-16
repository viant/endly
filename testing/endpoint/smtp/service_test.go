package smtp

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/notify/smtp"
	"github.com/viant/toolbox"
	"log"
	"path"
	"testing"
	"time"
)

func Test_Listen(t *testing.T) {
	parent := path.Join(toolbox.CallerDirectory(3), "test")
	credentialFile := path.Join(parent, "e2eendly.json")

	context := endly.New().NewContext(nil)
	err := endly.Run(context, &ListenRequest{
		ServerName:   "mail.e2eendly.com",
		Port:         1465,
		MaxBodySize:  1024 * 1024,
		CertLocation: parent,
		EnableTLS:    true,
		Users: []*User{
			{
				Credentials: credentialFile,
			},
			{
				Username: "john",
				Password: "somepassword",
			},
		},
	}, nil)
	if !assert.Nil(t, err) {
		log.Print(err)
		return
	}
	time.Sleep(2 * time.Second)
	err = endly.Run(context, &smtp.SendRequest{
		Target: location.NewResource("smtp://localhost:1465", credentialFile),
		Mail: &smtp.Message{
			From:    "tester@localhost",
			To:      []string{"bob@localhost"},
			Subject: "test subject",
			Body:    "this is test body",
		},
	}, nil)
	if !assert.Nil(t, err) {
		return
	}
	time.Sleep(1 * time.Second)
	fmt.Printf("\n")
	resp := &AssertResponse{}
	err = endly.Run(context, &AssertRequest{
		Expect: []*UserMessage{
			{
				User: "bob",
				Message: map[string]interface{}{
					"Subject": "test subject",
				},
			},
		},
	}, resp)
	if !assert.Equal(t, 1, len(resp.Validations)) {
		return
	}

	assert.Equal(t, 1, resp.Validations[0].PassedCount)
	if !assert.Equal(t, 0, resp.Validations[0].FailedCount) {
		log.Printf("%v", resp.Validations[0].Report())
	}
}
