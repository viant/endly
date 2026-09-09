//go:build !darwin

package ios

import (
	"fmt"
	"runtime"

	"github.com/viant/endly"
)

type unsupportedService struct {
	*endly.AbstractService
}

type unsupportedRouteSpec struct {
	action   string
	request  func() interface{}
	response func() interface{}
}

func New() endly.Service {
	result := &unsupportedService{AbstractService: endly.NewAbstractService(ServiceID)}
	result.AbstractService.Service = result
	routes := []unsupportedRouteSpec{
		{"doctor", func() interface{} { return &DoctorRequest{} }, func() interface{} { return &DoctorResponse{} }},
		{"simulator-start", func() interface{} { return &SimulatorStartRequest{} }, func() interface{} { return &SimulatorStartResponse{} }},
		{"simulator-stop", func() interface{} { return &SimulatorStopRequest{} }, func() interface{} { return &SimulatorStopResponse{} }},
		{"device-list", func() interface{} { return &DeviceListRequest{} }, func() interface{} { return &DeviceListResponse{} }},
		{"device-lease", func() interface{} { return &DeviceLeaseRequest{} }, func() interface{} { return &DeviceLeaseResponse{} }},
		{"device-release", func() interface{} { return &DeviceReleaseRequest{} }, func() interface{} { return &DeviceReleaseResponse{} }},
		{"destination-register", func() interface{} { return &DestinationRegisterRequest{} }, func() interface{} { return &DestinationRegisterResponse{} }},
		{"destination-release", func() interface{} { return &DestinationReleaseRequest{} }, func() interface{} { return &DestinationReleaseResponse{} }},
		{"server-start", func() interface{} { return &ServerStartRequest{} }, func() interface{} { return &ServerStartResponse{} }},
		{"server-stop", func() interface{} { return &ServerStopRequest{} }, func() interface{} { return &ServerStopResponse{} }},
		{"build", func() interface{} { return &BuildRequest{} }, func() interface{} { return &BuildResponse{} }},
		{"install", func() interface{} { return &InstallRequest{} }, func() interface{} { return &InstallResponse{} }},
		{"uninstall", func() interface{} { return &UninstallRequest{} }, func() interface{} { return &UninstallResponse{} }},
		{"launch", func() interface{} { return &LaunchRequest{} }, func() interface{} { return &LaunchResponse{} }},
		{"terminate", func() interface{} { return &TerminateRequest{} }, func() interface{} { return &TerminateResponse{} }},
		{"test", func() interface{} { return &TestRequest{} }, func() interface{} { return &TestResponse{} }},
		{"capture-start", func() interface{} { return &CaptureStartRequest{} }, func() interface{} { return &CaptureStartResponse{} }},
		{"capture-stop", func() interface{} { return &CaptureStopRequest{} }, func() interface{} { return &CaptureStopResponse{} }},
		{"open", func() interface{} { return &OpenRequest{} }, func() interface{} { return &OpenResponse{} }},
		{"attach", func() interface{} { return &AttachRequest{} }, func() interface{} { return &AttachResponse{} }},
		{"run", func() interface{} { return &RunRequest{} }, func() interface{} { return &RunResponse{} }},
		{"repl", func() interface{} { return &REPLRequest{} }, func() interface{} { return &REPLResponse{} }},
		{"artifact", func() interface{} { return &ArtifactRequest{} }, func() interface{} { return &ArtifactResponse{} }},
		{"close", func() interface{} { return &CloseRequest{} }, func() interface{} { return &CloseResponse{} }},
		{"cleanup", func() interface{} { return &CleanupRequest{} }, func() interface{} { return &CleanupResponse{} }},
	}
	for _, routeSpec := range routes {
		routeSpec := routeSpec
		action := routeSpec.action
		result.Register(&endly.Route{
			Action:           action,
			RequestInfo:      &endly.ActionInfo{Description: "iOS runner requires a macOS execution host"},
			RequestProvider:  routeSpec.request,
			ResponseProvider: routeSpec.response,
			Handler: func(_ *endly.Context, _ interface{}) (interface{}, error) {
				return nil, fmt.Errorf("ios:%s is unsupported on %s; run Endly on macOS or configure a remote macOS worker", action, runtime.GOOS)
			},
		})
	}
	return result
}
