package docker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	auth "github.com/docker/docker/api/types/registry"
	"github.com/viant/endly"
	"github.com/viant/scy/cred"
	"github.com/viant/scy/cred/secret"

	"strings"
)

// authConfigToken returns auth token
func authConfigToken(authConfig *auth.AuthConfig) (string, error) {
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

// authCredentialsToken returns auth token
func authCredentialsToken(context *endly.Context, credentials string) (string, error) {
	secret, err := context.Secrets.Lookup(context.Background(), secret.Resource(credentials))
	if err != nil {
		return "", err
	}
	generic, ok := secret.Target.(*cred.Generic)
	if !ok {
		return "", fmt.Errorf("unsupported secret type: %T, expected: %T", secret.Target, generic)
	}
	if generic.Username != "" && generic.Password != "" {
		return authConfigToken(&auth.AuthConfig{
			Username: generic.Username,
			Password: generic.Password,
		})
	}
	if generic.PrivateKeyID != "" {
		return authConfigToken(&auth.AuthConfig{
			Username: "_json_key",
			Password: strings.ReplaceAll(secret.String(), "\n", " "),
		})
	}
	return base64.URLEncoding.EncodeToString([]byte(secret.String())), nil
}

func getAuthToken(context *endly.Context, repository, credentials string) (string, error) {
	ctxClient, err := GetCtxClient(context)
	if err != nil {
		return "", err
	}
	if credentials == "" && ctxClient.AuthToken == nil {
		return "", nil
	}
	if credentials != "" {
		return authCredentialsToken(context, credentials)
	}
	return ctxClient.AuthToken[repository], nil
}
