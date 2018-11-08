package sso_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/dsc"
	"github.com/viant/endly/example/ui/sso"
	"github.com/viant/toolbox"
	"log"
	"path"
	"testing"
	"net/http"
)

func TestService_SignIn(t *testing.T) {
	var dataFileBaseDir = path.Join(toolbox.CallerDirectory(3), "test/data")
	var dataFile = path.Join(dataFileBaseDir, "users.json")
	toolbox.RemoveFileIfExist(dataFile)

	dsConfig := dsc.NewConfig("ndjson", "[url]", "dateFormat:yyyy-MM-dd hh:mm:ss,ext:json,url:"+toolbox.FileSchema+dataFileBaseDir)
	config := &sso.Config{
		Port: "8871",
		StaticRoutes: []*sso.StaticRoute{
			{
				URI:       "/signup/",
				Directory: "static/",
			},
			{
				URI:       "/signin/",
				Directory: "static/",
			},
		},
		DsConfig: dsConfig,
	}
	service, err := sso.NewService(config)
	if err != nil {
		log.Fatal(err)
	}
	{ //test singin with non existing users
		response := service.SignIn(&sso.SignInRequest{
			Email:    "abc@wp.pl",
			Password: "abc",
		})
		assert.Equal(t, "unable to find a user for provided credentials", response.Error)
	}
	{
		response := service.SignUp(&sso.SignUpRequest{
			User: &sso.User{
				Email: "abc@wp.pl",
				Name:  "abc",
			},
			Password:    "abc",
			DataOfBirth: "2002-02-11",
		}, &http.Request{RemoteAddr:"127.0.0.1"})

		assert.EqualValues(t, "", response.Error)
		assert.NotNil(t, response.User.DateOfBirth)
	}

	{ // try again register the same user
		response := service.SignUp(&sso.SignUpRequest{
			User: &sso.User{
				Email: "abc@wp.pl",
				Name:  "abc",
			},
			Password:    "abc",
			DataOfBirth: "2002-02-11",
		}, &http.Request{RemoteAddr:"127.0.0.1"})

		assert.EqualValues(t, "email abc@wp.pl has been already registered", response.Error)

	}

	{ //test singin with  existing users
		response := service.SignIn(&sso.SignInRequest{
			Email:    "abc@wp.pl",
			Password: "abc",
		})
		assert.Equal(t, "", response.Error)
		assert.NotNil(t, response.User)
	}

	{ //test singin with  invalid credentials users
		response := service.SignIn(&sso.SignInRequest{
			Email:    "abc@wp.pl",
			Password: "abc1",
		})
		assert.Equal(t, "unable to find a user for provided credentials", response.Error)
	}

}
