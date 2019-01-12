package gc

import (
	"context"
	"errors"
	"fmt"
	"github.com/viant/toolbox/cred"
	"golang.org/x/oauth2"
	"net/http"
)





//GetGCAuthClient creates a new compute service.
func GetGCAuthClient(ctx context.Context, credConfig *cred.Config, scopes ... string) (context.Context, *http.Client, error) {
	if credConfig == nil {
		return nil, nil, errors.New("credential config was empty")
	}
	if len(scopes) == 0 {
		return nil, nil, fmt.Errorf("scopes were empy")
	}
	jwtConfig, err := credConfig.NewJWTConfig(scopes...)
	if err != nil {
		return nil, nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	httpClient := oauth2.NewClient(ctx, jwtConfig.TokenSource(ctx))
	return ctx, httpClient, nil
}
