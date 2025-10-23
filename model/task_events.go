package model

// TaskStartEvent represents the start of a task within a workflow.
type TaskStartEvent struct {
	WorkflowName string   `json:",omitempty"`
	OwnerURL     string   `json:",omitempty"`
	TaskName     string   `json:",omitempty"`
	TaskPath     []string `json:",omitempty"`
	SessionID    string   `json:",omitempty"`
	Index        int      `json:",omitempty"`
	TemplateTag  *MetaTag `json:",omitempty"`
}

// NewTaskStartEvent creates a new TaskStartEvent.
func NewTaskStartEvent(workflowName, ownerURL, taskName string, taskPath []string, sessionID string, index int, templateTag *MetaTag) *TaskStartEvent {
	return &TaskStartEvent{
		WorkflowName: workflowName,
		OwnerURL:     ownerURL,
		TaskName:     taskName,
		TaskPath:     taskPath,
		SessionID:    sessionID,
		Index:        index,
		TemplateTag:  templateTag,
	}
}

// TaskEndEvent represents the end of a task within a workflow.
type TaskEndEvent struct {
	WorkflowName string   `json:",omitempty"`
	OwnerURL     string   `json:",omitempty"`
	TaskName     string   `json:",omitempty"`
	TaskPath     []string `json:",omitempty"`
	SessionID    string   `json:",omitempty"`
	Index        int      `json:",omitempty"`
	Status       string   `json:",omitempty"`
	Error        string   `json:",omitempty"`
}

// NewTaskEndEvent creates a new TaskEndEvent.
func NewTaskEndEvent(workflowName, ownerURL, taskName string, taskPath []string, sessionID string, index int, status, errMsg string) *TaskEndEvent {
	return &TaskEndEvent{
		WorkflowName: workflowName,
		OwnerURL:     ownerURL,
		TaskName:     taskName,
		TaskPath:     taskPath,
		SessionID:    sessionID,
		Index:        index,
		Status:       status,
		Error:        errMsg,
	}
}

// TaskAsyncStartEvent denotes that a task has launched asynchronous actions.
type TaskAsyncStartEvent struct {
	TaskPath  []string `json:",omitempty"`
	Expected  int      `json:",omitempty"`
	SessionID string   `json:",omitempty"`
}

// NewTaskAsyncStartEvent creates a new TaskAsyncStartEvent.
func NewTaskAsyncStartEvent(taskPath []string, expected int, sessionID string) *TaskAsyncStartEvent {
	return &TaskAsyncStartEvent{TaskPath: taskPath, Expected: expected, SessionID: sessionID}
}

// TaskAsyncDoneEvent denotes completion of asynchronous actions within a task.
type TaskAsyncDoneEvent struct {
	TaskPath  []string `json:",omitempty"`
	Completed int      `json:",omitempty"`
	SessionID string   `json:",omitempty"`
}

// NewTaskAsyncDoneEvent creates a new TaskAsyncDoneEvent.
func NewTaskAsyncDoneEvent(taskPath []string, completed int, sessionID string) *TaskAsyncDoneEvent {
	return &TaskAsyncDoneEvent{TaskPath: taskPath, Completed: completed, SessionID: sessionID}
}
