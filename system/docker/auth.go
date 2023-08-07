package docker

import (
	"encoding/base64"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/viant/endly"
	"strings"
)

// authConfigToken returns auth token
func authConfigToken(authConfig *types.AuthConfig) (string, error) {
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

// authCredentialsToken returns auth token
func authCredentialsToken(context *endly.Context, credentials string) (string, error) {
	cred, err := context.Secrets.GetCredentials(credentials)
	if err != nil {
		return "", err
	}
	if cred.Username != "" && cred.Password != "" {
		return authConfigToken(&types.AuthConfig{
			Username: cred.Username,
			Password: cred.Password,
		})
	}
	if cred.PrivateKeyID != "" {
		return authConfigToken(&types.AuthConfig{
			Username: "_json_key",
			Password: strings.Replace(cred.Data, "\n", " ", len(cred.Data)),
		})
	}
	return base64.URLEncoding.EncodeToString([]byte(cred.Data)), nil
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
