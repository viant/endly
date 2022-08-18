package bootstrap

import (
	"context"
	"github.com/viant/afsc/gs"
	"github.com/viant/scy/auth/gcp"
	"github.com/viant/scy/auth/gcp/client"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

func init() {
	srv := gcp.New(client.NewGCloud())
	gs.SetOptions(option.WithTokenSource(&tokenSource{Service: srv}))
}

type tokenSource struct {
	*gcp.Service
}

func (s *tokenSource) Token() (*oauth2.Token, error) {
	gcpScopes := append(gcp.Scopes, "https://www.googleapis.com/auth/bigquery")
	token, err := s.Auth(context.Background(), gcpScopes...)
	if err != nil {
		return nil, err
	}
	return &token.Token, nil
}
