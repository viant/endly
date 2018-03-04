package network

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/ssh"
)

const (
	//NetworkServiceID represents network service id
	NetworkServiceID = "network"

	//NetworkServiceTunnelAction represents opening ssh tunnel action
	NetworkServiceTunnelAction = "tunnel"
)

type service struct {
	*endly.AbstractService
}

func (s *service) tunnel(context *endly.Context, request *TunnelRequest) (*TunnelResponse, error) {
	var response = &TunnelResponse{
		Forwards: make([]*NetworkTunnel, 0),
	}
	var target, err = context.ExpandResource(request.Target)
	var authConfig = &cred.Config{}
	hostname, port := s.GetHostAndSSHPort(target)
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

func (s *service) registerRoutes() {
	s.Register(&endly.ServiceActionRoute{
		Action: "tunnel",
		RequestInfo: &endly.ActionInfo{
			Description: "tunnel tcp ports",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "tunnel",
					Data:    networkTunnelRequestExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &TunnelRequest{}
		},
		ResponseProvider: func() interface{} {
			return &TunnelResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*TunnelRequest); ok {
				return s.tunnel(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewCriteria creates a new network service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(NetworkServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
