package smtp

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox"
	"path"
	"testing"
)

func TestListenRequest_Init(t *testing.T) {

	parent := path.Join(toolbox.CallerDirectory(3), "test")
	credentialFile := path.Join(parent, "e2eendly.json")

	{
		listen := &ListenRequest{}
		assert.Nil(t, listen.Init())
		assert.Equal(t, "localhost", listen.ServerName)
		assert.Equal(t, 1587, listen.Port)
		assert.Equal(t, 1024*1024, listen.MaxBodySize)
	}

	{
		listen := &ListenRequest{
			ServerName:  "mail.me.com",
			Port:        587,
			MaxBodySize: 5 * 1024 * 1024,
			Users: []*User{
				{
					Username:    "john",
					Credentials: "zbc",
				},
				{
					Username:    "bob",
					Credentials: credentialFile,
				},
			},
		}
		assert.Nil(t, listen.Init())
		assert.Equal(t, "mail.me.com", listen.ServerName)
		assert.Equal(t, 587, listen.Port)
		assert.Equal(t, 5*1024*1024, listen.MaxBodySize)
		assert.True(t, listen.Users[1].Password != "")
	}

}

func TestListenRequest_Validate(t *testing.T) {
	{
		listen := &ListenRequest{}
		assert.NotNil(t, listen.Validate())
	}
	{
		listen := &ListenRequest{
			Users: []*User{
				{},
			},
		}
		assert.NotNil(t, listen.Validate())
	}
	{
		listen := &ListenRequest{
			Users: []*User{
				{
					Username: "bob",
					Password: "abc",
				},
			},
		}
		assert.Nil(t, listen.Validate())
	}
}
