package vc_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/service/deployment/vc"
	"github.com/viant/endly/internal/util"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/system/exec"
	"strings"
	"testing"
)

func TestVc_Status(t *testing.T) {
	credentialFile, err := util.GetDummyCredential()
	assert.Nil(t, err)
	var target = location.NewResource("ssh://127.0.0.1/Projects/project1/trunk", credentialFile) //

	var manager = endly.New()
	var useCases = []struct {
		baseDir  string
		Request  *vc.StatusRequest
		Expected *vc.Info
		Error    string
	}{
		{
			"test/svn/status/darwin",
			&vc.StatusRequest{
				Type:   "svn",
				Source: target,
			},
			&vc.Info{
				Untracked:               []string{".idea"},
				New:                     []string{"test.dep"},
				Modified:                []string{"pom.xml", "src/main/java/com/viant/Dummy.java"},
				Deleted:                 []string{"src/test/java/com/viant/DummyTest.java"},
				IsVersionControlManaged: true,
				Origin:                  "http://svn.viant.com/svn/projects/project1/trunk",
				Branch:                  "trunk",
			},

			"",
		},
	}

	for _, useCase := range useCases {
		var target = useCase.Request.Source
		context, err := exec.NewSSHReplayContext(manager, target, useCase.baseDir)
		if assert.Nil(t, err) {
			service, err := context.Service(vc.ServiceID)
			if assert.Nil(t, err) {
				serviceResponse := service.Run(context, useCase.Request)
				var description = useCase.baseDir + " "
				assert.Equal(t, useCase.Error, serviceResponse.Error, description)

				actual, ok := serviceResponse.Response.(*vc.StatusResponse)
				if !ok {
					assert.Fail(t, fmt.Sprintf("process serviceResponse was empty %v %T", description, serviceResponse.Response))
					continue
				}

				if actual == nil {
					continue
				}
				var expected = useCase.Expected
				assert.Equal(t, expected.Untracked, actual.Untracked, "Untracked "+description)
				assert.Equal(t, expected.New, actual.New, "New "+description)
				assert.Equal(t, expected.Modified, actual.Modified, "Modified "+description)
				assert.Equal(t, expected.Deleted, actual.Deleted, "Deleted "+description)
				assert.Equal(t, expected.IsVersionControlManaged, actual.IsVersionControlManaged, "IsVersionControlManaged "+description)
				assert.Equal(t, expected.IsUptoDate, actual.IsUptoDate, "IsUptoDate "+description)
				assert.Equal(t, expected.Origin, actual.Origin, "Origin "+description)
				assert.Equal(t, expected.Branch, actual.Branch, "Branch "+description)

			}
		}
	}
}

func TestVc_Checkout(t *testing.T) {
	credentialFile, err := util.GetCredential("localhost.json", "awitas", "***")
	gitCredentialFile, err := util.GetCredential("git.json", "adrianwit", "***")
	assert.Nil(t, err)

	var manager = endly.New()
	var useCases = []struct {
		baseDir  string
		Request  *vc.CheckoutRequest
		Expected *vc.CheckoutResponse
		Error    string
	}{
		{
			"test/svn/checkout/error/darwin",
			&vc.CheckoutRequest{
				Type:   "svn",
				Dest:   location.NewResource("scp://127.0.0.1:22/tmp/project2/trunk", credentialFile),
				Origin: location.NewResource("http://svn.viant.com/svn/projects/project1/trunk", credentialFile),
			},
			&vc.CheckoutResponse{},
			"failed to authenticate username: awitas with",
		},
		{
			"test/svn/checkout/new/darwin",
			&vc.CheckoutRequest{
				Type:   "svn",
				Dest:   location.NewResource("scp://127.0.0.1:22/tmp/project1/trunk", credentialFile),
				Origin: location.NewResource("http://svn.viant.com/svn/projects/project1/trunk", credentialFile),
			},
			&vc.CheckoutResponse{
				Checkouts: map[string]*vc.Info{

					"http://svn.viant.com/svn/projects/project1/trunk": {
						Origin:                  "http://svn.viant.com/svn/projects/project1/trunk",
						IsVersionControlManaged: true,
						IsUptoDate:              true,
						Modified:                []string{},
						Branch:                  "trunk",
					},
				},
			},
			"",
		},
		{
			"test/svn/checkout/existing/darwin",
			&vc.CheckoutRequest{
				Type:   "svn",
				Dest:   location.NewResource("scp://127.0.0.1:22/tmp/project1/trunk", credentialFile),
				Origin: location.NewResource("http://svn.viant.com/svn/projects/project1/trunk", credentialFile),
			},
			&vc.CheckoutResponse{
				Checkouts: map[string]*vc.Info{
					"http://svn.viant.com/svn/projects/project1/trunk": {
						Origin:                  "http://svn.viant.com/svn/projects/project1/trunk",
						IsVersionControlManaged: true,
						IsUptoDate:              false,
						Branch:                  "trunk",
						Modified:                []string{"pom.xml"},
					},
				},
			},
			"",
		},
		{
			"test/svn/checkout/modules/darwin",
			&vc.CheckoutRequest{
				Type:    "svn",
				Dest:    location.NewResource("scp://127.0.0.1:22/tmp/project3/", credentialFile),
				Origin:  location.NewResource("http://svn.viant.com/svn/projects/", credentialFile),
				Modules: []string{"project1/trunk", "project2/trunk"},
			},
			&vc.CheckoutResponse{
				Checkouts: map[string]*vc.Info{
					"http://svn.viant.com/svn/projects/project1/trunk": {
						Origin:                  "http://svn.viant.com/svn/projects/project1/trunk",
						IsVersionControlManaged: true,
						IsUptoDate:              true,
						Branch:                  "trunk",
						Modified:                []string{},
					},
					"http://svn.viant.com/svn/projects/project2/trunk": {
						Origin:                  "http://svn.viant.com/svn/projects/project2/trunk",
						IsVersionControlManaged: true,
						IsUptoDate:              true,
						Branch:                  "trunk",
						Modified:                []string{},
					},
				},
			},
			"",
		},
		{
			"test/git/checkout/private/error/linux",
			&vc.CheckoutRequest{

				Dest:   location.NewResource("scp://127.0.0.1:22/tmp/myproj", credentialFile),
				Origin: location.NewResource("https://github.com/adrianwit/projectA", gitCredentialFile),
			},
			&vc.CheckoutResponse{},
			"failed to authenticate username: adrianwit",
		},
		{
			"test/git/checkout/private/new/linux",
			&vc.CheckoutRequest{
				Dest:   location.NewResource("scp://127.0.0.1:22/tmp/myproj", credentialFile),
				Origin: location.NewResource("https://github.com/adrianwit/projectA", gitCredentialFile),
			},
			&vc.CheckoutResponse{
				Checkouts: map[string]*vc.Info{
					"https://github.com/adrianwit/projectA": {
						Origin:                  "https://adrianwit@github.com:443/adrianwit/projectA",
						IsVersionControlManaged: true,
						IsUptoDate:              true,
						Branch:                  "master",
						Modified:                []string{},
					},
				},
			},
			"",
		},
		{
			"test/git/checkout/private/existing/linux",
			&vc.CheckoutRequest{
				Dest:   location.NewResource("scp://127.0.0.1:22/tmp/myproj", credentialFile),
				Origin: location.NewResource("https://github.com/adrianwit/projectA", gitCredentialFile),
			},
			&vc.CheckoutResponse{
				Checkouts: map[string]*vc.Info{
					"https://github.com/adrianwit/projectA": {
						Origin:                  "https://adrianwit@github.com:443/adrianwit/projectA",
						IsVersionControlManaged: true,
						IsUptoDate:              false,
						Branch:                  "master",
						Modified:                []string{"README.md"},
					},
				},
			},
			"",
		},
	}

	for i, useCase := range useCases {
		var target = useCase.Request.Dest
		context, err := exec.NewSSHReplayContext(manager, target, useCase.baseDir)
		if assert.Nil(t, err) {
			service, err := context.Service(vc.ServiceID)
			if assert.Nil(t, err) {

				serviceResponse := service.Run(context, useCase.Request)
				var description = useCase.baseDir + " " + fmt.Sprintf("[%d]", i)
				if useCase.Error != "" {
					assert.True(t, strings.Contains(serviceResponse.Error, useCase.Error), description+" "+serviceResponse.Error)
				}
				response, ok := serviceResponse.Response.(*vc.CheckoutResponse)
				if !ok {
					assert.Fail(t, fmt.Sprintf("process serviceResponse was empty %v %T", description, serviceResponse.Response))
					continue
				}

				if response == nil {
					continue
				}
				assert.EqualValues(t, useCase.Error, serviceResponse.Error)
				if useCase.Error != "" {
					continue
				}
				for key, actual := range response.Checkouts {

					expected, ok := useCase.Expected.Checkouts[key]
					if !assert.True(t, ok, "missing origin: "+key) {
						continue
					}
					assert.Equal(t, expected.IsVersionControlManaged, actual.IsVersionControlManaged, "IsVersionControlManaged "+description)
					assert.Equal(t, expected.IsUptoDate, actual.IsUptoDate, "IsUptoDate "+description)
					assert.Equal(t, expected.Origin, actual.Origin, "Origin "+description)
					assert.Equal(t, expected.Branch, actual.Branch, "Branch "+description)
					assert.Equal(t, expected.Modified, actual.Modified, "Modified "+description)
				}

			}
		}
	}
}
