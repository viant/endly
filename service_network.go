package endly

import (
	"fmt"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/ssh"
)

//NetworkServiceID represents java network service id
const NetworkServiceID = "network"

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

func (s *networkService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	switch actualRequest := request.(type) {
	case *NetworkTunnelRequest:
		response.Response, err = s.tunnel(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run tunnel: %v, %v", actualRequest.Target.URL, err)
		}
	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

//NewRequest creates a new request for an action (run).
func (s *networkService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "tunnel":
		return &NetworkTunnelRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewNetworkService creates a new network service.
func NewNetworkService() Service {
	var result = &networkService{
		AbstractService: NewAbstractService(NetworkServiceID),
	}
	result.AbstractService.Service = result
	return result
}
