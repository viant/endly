package util

import (
	"context"
	"fmt"
	"github.com/viant/scy/cred"
	"github.com/viant/scy/cred/secret"
)




func GetUsername(service *secret.Service, credentials string) (string, error) {
	var username string
	secret, err := service.Lookup(context.Background(), secret.Resource(credentials))
	if err != nil {
		return "", err
	}
	generic, ok := secret.Target.(*cred.Generic)
	if ! ok {
		return "", fmt.Errorf("unsupported secret type: %T, expected: %T", secret.Target, generic)
	}
	username = generic.Username
	if username == "" {
		return "", fmt.Errorf("username was empty %v", credentials)
	}
	return username, nil
}
