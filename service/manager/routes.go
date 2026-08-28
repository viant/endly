package manager

import (
	"fmt"

	"github.com/viant/endly"
)

func (s *Service) registerRoutes() {
	s.Register(
		s.route("open", func() interface{} { return &OpenRequest{} }, func() interface{} { return &SessionInfo{} }, func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.Open(ctx.Background(), request.(*OpenRequest))
		}),
		s.route("listSessions", func() interface{} { return &struct{}{} }, func() interface{} { return &ListSessionsResponse{} }, func(_ *endly.Context, _ interface{}) (interface{}, error) {
			return s.Sessions(), nil
		}),
		s.route("close", func() interface{} { return &CloseRequest{} }, func() interface{} { return &struct{}{} }, func(ctx *endly.Context, request interface{}) (interface{}, error) {
			err := s.Close(ctx.Background(), request.(*CloseRequest).SessionID)
			return &struct{}{}, err
		}),
		s.route("loadWorkflow", func() interface{} { return &LoadWorkflowRequest{} }, func() interface{} { return &LoadedWorkflow{} }, func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.LoadWorkflow(ctx.Background(), request.(*LoadWorkflowRequest))
		}),
		s.route("listWorkflows", func() interface{} { return &ListWorkflowsRequest{} }, func() interface{} { return &ListWorkflowsResponse{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			return s.ListWorkflows(request.(*ListWorkflowsRequest).SessionID)
		}),
		s.route("unloadWorkflow", func() interface{} { return &UnloadWorkflowRequest{} }, func() interface{} { return &struct{}{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			err := s.UnloadWorkflow(request.(*UnloadWorkflowRequest))
			return &struct{}{}, err
		}),
		s.route("runWorkflow", func() interface{} { return &RunWorkflowRequest{} }, func() interface{} { return &Operation{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			return s.StartWorkflow(request.(*RunWorkflowRequest))
		}),
		s.route("startWorkflow", func() interface{} { return &RunWorkflowRequest{} }, func() interface{} { return &Operation{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			return s.StartWorkflow(request.(*RunWorkflowRequest))
		}),
		s.route("runAction", func() interface{} { return &RunActionRequest{} }, func() interface{} { return &Operation{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			return s.StartAction(request.(*RunActionRequest))
		}),
		s.route("getOperation", func() interface{} { return &GetOperationRequest{} }, func() interface{} { return &Operation{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			r := request.(*GetOperationRequest)
			return s.GetOperation(r.SessionID, r.OperationID)
		}),
		s.route("listOperations", func() interface{} { return &ListOperationsRequest{} }, func() interface{} { return &ListOperationsResponse{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			return s.ListOperations(request.(*ListOperationsRequest).SessionID)
		}),
		s.route("stopOperation", func() interface{} { return &StopOperationRequest{} }, func() interface{} { return &Operation{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			r := request.(*StopOperationRequest)
			return s.StopOperation(r.SessionID, r.OperationID)
		}),
		s.route("inspectContext", func() interface{} { return &InspectContextRequest{} }, func() interface{} { return &ContextInspection{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			return s.InspectContext(request.(*InspectContextRequest))
		}),
		s.route("setLogging", func() interface{} { return &SetLoggingRequest{} }, func() interface{} { return &LoggingState{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			return s.SetLogging(request.(*SetLoggingRequest))
		}),
		s.route("getLogging", func() interface{} { return &GetLoggingRequest{} }, func() interface{} { return &LoggingState{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			return s.Logging(request.(*GetLoggingRequest).SessionID)
		}),
		s.route("getDebugState", func() interface{} { return &GetOperationRequest{} }, func() interface{} { return &DebugState{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			r := request.(*GetOperationRequest)
			return s.DebugState(r.SessionID, r.OperationID)
		}),
		s.route("debugCommand", func() interface{} { return &DebugCommandRequest{} }, func() interface{} { return &DebugState{} }, func(_ *endly.Context, request interface{}) (interface{}, error) {
			return s.DebugCommand(request.(*DebugCommandRequest))
		}),
	)
}

func (s *Service) route(action string, requestProvider, responseProvider func() interface{}, handler func(*endly.Context, interface{}) (interface{}, error)) *endly.Route {
	return &endly.Route{
		Action: action,
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("stateful manager %s", action),
		},
		RequestProvider:  requestProvider,
		ResponseProvider: responseProvider,
		Handler:          handler,
	}
}
