package endly

import (
	"fmt"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/ssh"
)

const (
	//NetworkServiceID represents network service id
	NetworkServiceID = "network"

	//NetworkServiceTunnelAction represents opening ssh tunnel action
	NetworkServiceTunnelAction = "tunnel"
)

type networkService struct {
	*AbstractService
}

func (s *networkService) tunnel(context *Context, request *NetworkTunnelRequest) (*NetworkTunnelResponse, error) {
	var response = &NetworkTunnelResponse{
		Forwards: make([]*NetworkTunnel, 0),
	}
	var target, err = context.ExpandResource(request.Target)
	var authConfig = &cred.Config{}
	hostname, port := getHostAndSSHPort(target)
	client, err := ssh.NewService(hostname, port, authConfig)
	if err != nil {
		return nil, err
	}
	context.Deffer(func() {
		client.Close()
	})
	for _, tunnel := range request.Tunnels {
		var local = context.Expand(tunnel.Local)
		var remote = context.Expand(tunnel.Remote)
		err = client.OpenTunnel(local, remote)
		if err != nil {
			return nil, err
		}
		response.Forwards = append(response.Forwards, &NetworkTunnel{Local: local, Remote: remote})
	}
	return response, nil
}

const networkTunnelRequestExample = `{
	"Local":"127.0.0.1:8080",
	"Remote":"127.0.0.1:8080"
}
`

func (s *networkService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "tunnel",
		RequestInfo: &ActionInfo{
			Description: "tunnel tcp ports",
			Examples: []*ExampleUseCase{
				{
					UseCase: "tunnel",
					Data:    networkTunnelRequestExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &NetworkTunnelRequest{}
		},
		ResponseProvider: func() interface{} {
			return &NetworkTunnelResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*NetworkTunnelRequest); ok {
				return s.tunnel(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewNetworkService creates a new network service.
func NewNetworkService() Service {
	var result = &networkService{
		AbstractService: NewAbstractService(NetworkServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
