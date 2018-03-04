package util

import (
	"os"
	"path"
	"fmt"
	"time"
	"github.com/viant/toolbox/cred"
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

