package vc

import (
	"errors"
	"fmt"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox/secret"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
)

func versionControlRequestInit(resource *url.Resource, vcType *string) error {
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

var secretService = secret.New("", false)

var errorRewrites = map[string]func(*secret.Service, *url.Resource) string{
	"authentication failed": func(service *secret.Service, resource *url.Resource) string {
		username, _ := util.GetUsername(service, resource.Credentials)
		return fmt.Sprintf("failed to authenticate username: %v with %v secret", username, resource.Credentials)
	},
	"error validating server certificate": func(service *secret.Service, resource *url.Resource) string {
		return fmt.Sprintf("failed to validate svn certificate: %v", resource.URL)
	},
	"username": func(service *secret.Service, resource *url.Resource) string {
		username, _ := util.GetUsername(service, resource.Credentials)
		return fmt.Sprintf("failed to authenticate username: %v with %v secret", username, resource.Credentials)
	},
}

func checkVersionControlAuthErrors(err error, service *secret.Service, resource *url.Resource) error {
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
	var resource = url.NewResource(URL)
	var result = resource.ParsedURL.Hostname() + resource.ParsedURL.Path
	ext := path.Ext(result)
	if ext != "" {
		result = string(result[:len(result)-len(ext)])
	}
	return result
}
