package util

import (
	"context"
	"fmt"

	"github.com/viant/scy"
	"github.com/viant/scy/cred"
	"github.com/viant/scy/cred/secret"
)

// SecretLookup is the minimal secret API used by util helpers.
type SecretLookup interface {
	Lookup(ctx context.Context, resource secret.Resource) (*scy.Secret, error)
}

func GetUsername(service SecretLookup, credentials string) (string, error) {
	var username string
	secret, err := service.Lookup(context.Background(), secret.Resource(credentials))
	if err != nil {
		return "", err
	}
	generic, ok := secret.Target.(*cred.Generic)
	if !ok {
		return "", fmt.Errorf("unsupported secret type: %T, expected: %T", secret.Target, generic)
	}
	username = generic.Username
	if username == "" {
		return "", fmt.Errorf("username was empty %v", credentials)
	}
	return username, nil
}
