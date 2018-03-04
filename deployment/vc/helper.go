package vc

import (
	"errors"
	"fmt"
	"github.com/viant/toolbox/url"
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

var errorRewrites = map[string]func(*url.Resource) string{
	"authentication failed": func(resource *url.Resource) string {
		username, _, _ := resource.LoadCredential(false)
		return fmt.Sprintf("failed to authenticate username: %v with %v secret", username, resource.Credential)
	},
	"error validating server certificate": func(resource *url.Resource) string {
		return fmt.Sprintf("failed to validate svn certificate: %v", resource.URL)
	},
	"username": func(resource *url.Resource) string {
		username, _, _ := resource.LoadCredential(false)
		return fmt.Sprintf("failed to authenticate username: %v with %v secret", username, resource.Credential)
	},
}

func checkVersionControlAuthErrors(err error, resource *url.Resource) error {
	if err != nil {
		errorMessage := strings.ToLower(err.Error())
		for candidate, callback := range errorRewrites {
			if strings.Contains(errorMessage, candidate) {
				return errors.New(callback(resource))
			}
		}
	}
	return err
}
