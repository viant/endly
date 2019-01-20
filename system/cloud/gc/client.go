package gc

import (
	"context"
)

//CtxClient represents generic google cloud service client
type CtxClient interface {
	SetContext(ctx context.Context)
	Context() context.Context
	SetService(client interface{}) error
	Service() interface{}
}

//AbstractClient represents an abstract client
type AbstractClient struct {
	context context.Context
	scopes  []string
}

func (c *AbstractClient) SetContext(ctx context.Context) {
	c.context = ctx
}

func (c *AbstractClient) Context() context.Context {
	return c.context
}
