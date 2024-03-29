package git

import (
	"github.com/viant/endly"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	ssh2 "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

func getAuth(context *endly.Context, credentials string) (transport.AuthMethod, error) {
	credConifg, err := context.Secrets.GetCredentials(context.Background(), credentials)
	if err != nil {
		return nil, err
	}
	if credConifg.PrivateKeyPath != "" {
		sshAuth, err := ssh2.NewPublicKeysFromFile("git", credConifg.PrivateKeyPath, credConifg.Password)
		if err != nil {
			return nil, err
		}
		return sshAuth, nil
	}
	return &http.BasicAuth{
		Username: credConifg.Username,
		Password: credConifg.Password,
	}, nil
}
