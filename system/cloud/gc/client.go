package gc

import (
	"context"
	"net/http"
	"github.com/viant/toolbox/cred"
)

//CtxClient represents generic google cloud service client
type CtxClient interface {
	SetContext(ctx context.Context)
	Context() context.Context
	SetService(client interface{}) error
	Service() interface{}
	SetCredConfig(config *cred.Config)
	SetHttpClient(client *http.Client)

}

//AbstractClient represents an abstract client
type AbstractClient struct {
	context    context.Context
	scopes     []string
	CredConfig *cred.Config
	HttpClinet *http.Client
}

func (c *AbstractClient) SetContext(ctx context.Context) {
	c.context = ctx
}

func (c *AbstractClient) Context() context.Context {
	return c.context
}

func (c *AbstractClient) SetCredConfig(config *cred.Config) {
	c.CredConfig = config
}


func (c *AbstractClient) SetHttpClient(client *http.Client) {
	c.HttpClinet = client
}
