package bigquery

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"google.golang.org/api/bigquery/v2"
	"log"
	"time"
)

const (
	//ServiceID Google BigQuery Service ID.
	ServiceID  = "gcp/bigquery"
	doneStatus = "DONE"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) load(context *endly.Context, request *LoadRequest) (*bigquery.Job, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	jobService := bigquery.NewJobsService(client.service)

	insertCall := jobService.Insert(request.Project, &bigquery.Job{
		Configuration: &bigquery.JobConfiguration{
			Load: &request.JobConfigurationLoad,
		},
		JobReference: request.Job,
	})

	job, err := insertCall.Do()
	if request.Async || err != nil {
		return job, err
	}
	return s.jobWait(context, &JobWaitRequest{
		Job: job.JobReference,
	})
}

func (s *service) query(context *endly.Context, request *QueryRequest) (*bigquery.Job, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	jobService := bigquery.NewJobsService(client.service)

	insertCall := jobService.Insert(request.Project, &bigquery.Job{
		Configuration: &bigquery.JobConfiguration{
			Query: &request.JobConfigurationQuery,
		},
		JobReference: request.Job,
	})

	job, err := insertCall.Do()
	if request.Async || err != nil {
		return job, err
	}
	return s.jobWait(context, &JobWaitRequest{
		Job: job.JobReference,
	})
}

func (s *service) copy(context *endly.Context, request *CopyRequest) (*bigquery.Job, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	jobService := bigquery.NewJobsService(client.service)

	insertCall := jobService.Insert(request.Project, &bigquery.Job{
		Configuration: &bigquery.JobConfiguration{
			Copy: &request.JobConfigurationTableCopy,
		},
		JobReference: request.Job,
	})

	job, err := insertCall.Do()
	if request.Async || err != nil {
		return job, err
	}
	return s.jobWait(context, &JobWaitRequest{
		Job: job.JobReference,
	})
}

func (s *service) jobWait(context *endly.Context, request *JobWaitRequest) (response *bigquery.Job, err error) {
	err = s.RunInBackground(context, func() error {
		response, err = s.waitForOperationCompletion(context, request.Job)
		return err
	})
	return response, err
}

func (s *service) waitForOperationCompletion(context *endly.Context, job *bigquery.JobReference) (*bigquery.Job, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	service := bigquery.NewJobsService(client.service)
	for {
		getCall := service.Get(job.ProjectId, job.JobId)
		getCall.Context(client.Context())
		job, err := getCall.Do()
		if err != nil {
			return nil, err
		}
		if job.Status.State == doneStatus {
			return job, err
		}
		time.Sleep(time.Second)
	}
}

func (s *service) registerRoutes() {
	client := &bigquery.Service{}
	routes, err := gcp.BuildRoutes(client, nil, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = InitRequest
		s.Register(route)
	}

	s.Register(&endly.Route{
		Action: "query",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "query", &QueryRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &bigquery.Job{}),
		},
		RequestProvider: func() interface{} {
			return &QueryRequest{}
		},
		ResponseProvider: func() interface{} {
			return &bigquery.Job{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*QueryRequest); ok {
				output, err := s.query(context, req)
				if err != nil {
					return nil, err
				}
				context.Publish(gcp.NewOutputEvent("...", "list", output))
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "load",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "load", &LoadRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &bigquery.Job{}),
		},
		RequestProvider: func() interface{} {
			return &LoadRequest{}
		},
		ResponseProvider: func() interface{} {
			return &bigquery.Job{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*LoadRequest); ok {
				output, err := s.load(context, req)
				if err != nil {
					return nil, err
				}
				context.Publish(gcp.NewOutputEvent("...", "list", output))
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "copy",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "copy", &CopyRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &bigquery.Job{}),
		},
		RequestProvider: func() interface{} {
			return &CopyRequest{}
		},
		ResponseProvider: func() interface{} {
			return &bigquery.Job{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CopyRequest); ok {
				output, err := s.copy(context, req)
				if err != nil {
					return nil, err
				}
				context.Publish(gcp.NewOutputEvent("...", "list", output))
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "jobWait",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "jobWait", &JobWaitRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &bigquery.Job{}),
		},
		RequestProvider: func() interface{} {
			return &JobWaitRequest{}
		},
		ResponseProvider: func() interface{} {
			return &bigquery.Job{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*JobWaitRequest); ok {
				output, err := s.jobWait(context, req)
				if err != nil {
					return nil, err
				}
				context.Publish(gcp.NewOutputEvent("...", "list", output))
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new BigQuery service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
