package gcp

import (
	"context"
	"github.com/viant/toolbox/cred"
	"net/http"
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
	service    interface{}
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


func (c *AbstractClient) SetService(service interface{}) error {
	c.service = service
	return nil
}

func (c *AbstractClient) Service() interface{} {
	return c.service
}
