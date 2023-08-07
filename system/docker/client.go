package docker

import (
	"github.com/docker/docker/client"
	"github.com/viant/endly"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"golang.org/x/net/context"
)

var clientKey = (*CtxClient)(nil)

// CtxClient represents generic docker client
type CtxClient struct {
	Client     *client.Client
	Context    context.Context
	APIVersion string
	AuthToken  map[string]string
}

// GetCtxClient get or creates a new  kubernetess client.
func GetCtxClient(ctx *endly.Context) (*CtxClient, error) {
	result := &CtxClient{}
	if ctx.Contains(clientKey) {
		if ctx.GetInto(clientKey, &result) {
			return result, nil
		}
	}

	var err error
	if len(result.AuthToken) == 0 {
		result.AuthToken = make(map[string]string)
	}
	result.Context = context.Background()
	if result.APIVersion == "" {
		result.APIVersion = "1.37"
	}

	//TODO extends CtxClient with global parameters to enable https client and other types
	if result.Client, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion(result.APIVersion)); err != nil {
		return nil, err
	}
	err = ctx.Replace(clientKey, result)
	return result, err
}

// initClient get or creates context client
func initClient(context *endly.Context, rawRequest map[string]interface{}) error {
	if len(rawRequest) == 0 {
		return nil
	}
	ctxClient, err := GetCtxClient(context)
	if err != nil {
		return err
	}
	mappings := util.BuildLowerCaseMapping(rawRequest)
	if key, ok := mappings["apiversion"]; ok {
		ctxClient.APIVersion = toolbox.AsString(rawRequest[key])
		ctxClient.Client, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion(ctxClient.APIVersion))
	}
	return err
}
