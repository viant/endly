package network

import (
	"fmt"
	"github.com/viant/endly"
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
	if err != nil {
		return nil, err
	}
	authConfig, err := context.Secrets.GetCredentials(target.Credentials)
	if err != nil {
		return nil, err
	}
	hostname, port := s.GetHostAndSSHPort(target)
	sshClient, err := ssh.NewService(hostname, port, authConfig)
	if err != nil {
		return nil, err
	}
	context.Deffer(func() {
		sshClient.Close()
	})
	for _, tunnel := range request.Tunnels {
		var local = context.Expand(tunnel.Local)
		var remote = context.Expand(tunnel.Remote)
		err = sshClient.OpenTunnel(local, remote)
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
	s.Register(&endly.Route{
		Action: "tunnel",
		RequestInfo: &endly.ActionInfo{
			Description: "tunnel tcp ports",
			Examples: []*endly.UseCase{
				{
					Description: "tunnel",
					Data:        networkTunnelRequestExample,
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

//New creates a new network service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(NetworkServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
