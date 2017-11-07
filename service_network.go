package endly

import (
	"fmt"
	"github.com/viant/toolbox/ssh"
	"github.com/viant/toolbox/cred"
)


//NetworkServiceID represents java network service id
const NetworkServiceID = "network"



type networkService struct {
	*AbstractService
}

func (s *networkService) forward(context *Context, request *NetworkForwardRequest) (*NetworkForwardResponse, error) {
	var response = &NetworkForwardResponse{
		Forwards: make([]*NetworkForward, 0),
	}
	var target, err = context.ExpandResource(request.Target)
	var authConfig = &cred.Config{}
	hostname, port := getHostAndSSHPort(target)
	client, err := ssh.NewClient(hostname, port, authConfig)
	if err != nil {
		return nil, err
	}
	context.Deffer(func() {
		client.Close()
	})
	for _, forward := range request.Forwards {
		var local = context.Expand(forward.Local)
		var remote = context.Expand(forward.Remote)

		fmt.Printf(" %v -> %v\n", local, remote)
		err = client.Forward(local, remote)
		if err != nil {
			return nil, err
		}
		response.Forwards = append(response.Forwards, &NetworkForward{Local:local, Remote:remote})
	}
	return response, nil
}



func (s *networkService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	switch actualRequest := request.(type) {
	case *NetworkForwardRequest:
		response.Response, err = s.forward(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run forward: %v, %v", actualRequest.Target.URL, err)
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
	case "forward":
		return &NetworkForwardRequest{}, nil
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

