package bigquery

import (
	"errors"
	"github.com/viant/endly"
	"google.golang.org/api/bigquery/v2"
)

// LoadRequest represents load request
type LoadRequest struct {
	bigquery.JobConfigurationLoad
	Job     *bigquery.JobReference
	Project string
	Async   bool `description:"if set true, function does not wait for job completion"`
}

// Validate checks if request is valid
func (r *LoadRequest) Validate() error {
	if r.DestinationTable == nil {
		return errors.New("destination table was empty")
	}
	return nil
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
