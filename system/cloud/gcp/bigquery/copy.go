package bigquery

import (
	"errors"
	"github.com/viant/endly"
	"google.golang.org/api/bigquery/v2"
)

//CopyRequest represents copy request
type CopyRequest struct {
	bigquery.JobConfigurationTableCopy
	Project string
	Job     *bigquery.JobReference
	Async   bool `description:"if set true, function does not wait for job completion"`
}


//Validate checks if request is valid
func (r *CopyRequest) Validate() error {
	if r.DestinationTable == nil {
		return errors.New("destination table was empty")
	}
	if r.SourceTable == nil {
		return errors.New("source table was empty")
	}
	return nil
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

