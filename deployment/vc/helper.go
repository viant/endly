package vc

import (
	"errors"
	"fmt"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/util"
	"github.com/viant/scy/cred/secret"
	"path"
	"strings"
)

func versionControlRequestInit(resource *location.Resource, vcType *string) error {
	if resource == nil {
		return nil
	}
	if *vcType == "" {
		if strings.Contains(resource.URL, "/svn") {
			*vcType = "svn"
		} else if strings.Contains(resource.URL, "git") {
			*vcType = "git"
		}
	}
	return nil
}

var errorRewrites = map[string]func(*secret.Service, *location.Resource) string{
	"authentication failed": func(service *secret.Service, resource *location.Resource) string {
		username, _ := util.GetUsername(service, resource.Credentials)
		return fmt.Sprintf("failed to authenticate username: %v with %v secret", username, resource.Credentials)
	},
	"error validating server certificate": func(service *secret.Service, resource *location.Resource) string {
		return fmt.Sprintf("failed to validate svn certificate: %v", resource.URL)
	},
	"username": func(service *secret.Service, resource *location.Resource) string {
		username, _ := util.GetUsername(service, resource.Credentials)
		return fmt.Sprintf("failed to authenticate username: %v with %v secret", username, resource.Credentials)
	},
}

func checkVersionControlAuthErrors(err error, service *secret.Service, resource *location.Resource) error {
	if err != nil {
		errorMessage := strings.ToLower(err.Error())
		for candidate, callback := range errorRewrites {
			if strings.Contains(errorMessage, candidate) {
				return errors.New(callback(service, resource))
			}
		}
	}
	return err
}

func normalizeVCPath(URL string) string {
	var resource = location.NewResource(URL)
	var result = resource.Hostname() + resource.Path()
	ext := path.Ext(result)
	if ext != "" {
		result = string(result[:len(result)-len(ext)])
	}
	return result
}
