package bigquery

import (
	"errors"
	"github.com/viant/endly"
	"google.golang.org/api/bigquery/v2"
)


//QueryRequest represents query request
type QueryRequest struct {
	bigquery.JobConfigurationQuery
	Job     *bigquery.JobReference
	Project string
	Async   bool `description:"if set true, function does not wait for job completion"`
}

//Validate checks if request is valid
func (r *QueryRequest) Validate() error {
	if r.Query == "" {
		return errors.New("query was empty")
	}
	return nil
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
