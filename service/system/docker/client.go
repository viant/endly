package docker

import (
	"strconv"
	"strings"

	"github.com/docker/docker/client"
	"github.com/viant/endly"
	"github.com/viant/endly/internal/util"
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
	// Prefer API version negotiation with the daemon to avoid hard-coding.
	// If APIVersion is set later via initClient, we will recreate the client accordingly.
	if result.Client, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err != nil {
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
		// Ensure minimum supported version 1.44 to prevent daemon rejection.
		parts := strings.Split(ctxClient.APIVersion, ".")
		if len(parts) == 2 {
			if minor, convErr := strconv.Atoi(parts[1]); convErr == nil {
				if minor < 44 {
					ctxClient.APIVersion = "1.44"
				}
			}
		}
		ctxClient.Client, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion(ctxClient.APIVersion))
		return err
	}
	// If no explicit version provided, recreate with negotiation to pick latest compatible.
	ctxClient.Client, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	return err
}
