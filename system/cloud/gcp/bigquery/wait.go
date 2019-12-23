package bigquery

import (
	"errors"
	"github.com/viant/endly"
	"google.golang.org/api/bigquery/v2"
	"time"
)

type JobWaitRequest struct {
	Job *bigquery.JobReference
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



//Validate checks if request is valid
func (r *JobWaitRequest) Validate() error {
	if r.Job == nil {
		return errors.New("job was empty")
	}
	return nil
}
