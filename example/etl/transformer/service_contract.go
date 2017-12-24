package transformer

import (
	"github.com/viant/dsc"
	"time"
)

const (
	//StatusTaskNotRunning  represent terminated task
	StatusTaskNotRunning = iota
	//StatusTaskRunning represents active copy task
	StatusTaskRunning
)

//BaseResponse represents a base response
type BaseResponse struct {
	Status    string
	Error     string
	StartTime time.Time
	EndTime   time.Time
}

//DatasetResource represents a datastore resource
type DatasetResource struct {
	DsConfig  *dsc.Config
	Table     string
	PkColumns []string
	Columns   []string
	SQL       string
}

//AsTableDescription converts data resource as table descriptor
func (r *DatasetResource) AsTableDescription() *dsc.TableDescriptor {
	return &dsc.TableDescriptor{
		Table:     r.Table,
		Columns:   r.Columns,
		PkColumns: r.PkColumns,
	}
}

//TaskInfo represents processed record info
type TaskInfo struct {
	Status             string
	StatusCode         int32
	SkippedRecordCount int
	EmptyRecordCount   int
	RecordCount        int
}

//CopyRequest represents a copy request
type CopyRequest struct {
	BatchSize   int
	InsertMode  bool
	Source      *DatasetResource
	Destination *DatasetResource
	Transformer string
}

//CopyResponse represents a copy response
type CopyResponse struct {
	*BaseResponse
	*TaskInfo
}

//TaskListRequest represents a task list request
type TaskListRequest struct {
	Table string
}

//Task represents a task
type Task struct {
	ID         string
	Status     string
	StatusCode int32
	Table      string
	Request    interface{}
	*BaseResponse
	*TaskInfo
}

//Expired returns true if task expired
func (t *Task) Expired(currentTime time.Time) bool {
	if !t.EndTime.IsZero() {
		return currentTime.Sub(t.EndTime) > time.Hour
	}
	return false
}

//TaskListResponse represents task list response
type TaskListResponse struct {
	Status string
	Tasks  []*Task
}

//KillTaskRequest represents kill task
type KillTaskRequest struct {
	ID string
}

//KillTaskResponse represents kill task response
type KillTaskResponse struct {
	*BaseResponse
	Task *Task
}
