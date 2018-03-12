package util

import (
	"fmt"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/secret"
	"os"
	"path"
	"time"
)

func GetDummyCredential() (string, error) {
	return GetCredential("dummy", os.Getenv("USER"), "***")
}

func GetCredential(name, username, password string) (string, error) {
	var credentialFile = path.Join(os.TempDir(), fmt.Sprintf("%v%v.json", name, time.Now().Hour()))
	authConfig := cred.Config{
		Username: username,
		Password: password,
	}
	err := authConfig.Save(credentialFile)
	return credentialFile, err
}

func GetUsername(service *secret.Service, credential string) (string, error) {
	var username string
	credConfig, err := service.GetCredentials(credential)
	if err != nil {
		return "", err
	}
	username = credConfig.Username
	if username == "" {
		return "", fmt.Errorf("username was empty %v", credential)
	}
	return username, nil
}
