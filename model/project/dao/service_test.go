package dao

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestIt(t *testing.T) {

	ctx := context.Background()
	srv, err := New(ctx)
	if !assert.Nil(t, err) {
		return
	}
	err = srv.Init(ctx)
	if !assert.Nil(t, err) {
		return
	}
	handler, err := srv.Service.HTTPHandler(ctx)
	if !assert.Nil(t, err) {
		return
	}
	assert.NotNil(t, handler)
	http.ListenAndServe(":8081", handler)

}
