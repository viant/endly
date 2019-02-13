package bigquery

import (
	"errors"
	"google.golang.org/api/bigquery/v2"
)

//CopyRequest represents copy request
type CopyRequest struct {
	bigquery.JobConfigurationTableCopy
	Project string
	Job     *bigquery.JobReference
	Async   bool `description:"if set true, function does not wait for job completion"`
}

//LoadRequest represents load request
type LoadRequest struct {
	bigquery.JobConfigurationLoad
	Job     *bigquery.JobReference
	Project string
	Async   bool `description:"if set true, function does not wait for job completion"`
}

//QueryRequest represents query request
type QueryRequest struct {
	bigquery.JobConfigurationQuery
	Job     *bigquery.JobReference
	Project string
	Async   bool `description:"if set true, function does not wait for job completion"`
}

type JobWaitRequest struct {
	Job *bigquery.JobReference
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

//Validate checks if request is valid
func (r *QueryRequest) Validate() error {
	if r.Query == "" {
		return errors.New("query was empty")
	}
	return nil
}

//Validate checks if request is valid
func (r *LoadRequest) Validate() error {
	if r.DestinationTable == nil {
		return errors.New("destination table was empty")
	}
	return nil
}

//Validate checks if request is valid
func (r *JobWaitRequest) Validate() error {
	if r.Job == nil {
		return errors.New("job was empty")
	}
	return nil
}
