package workflow

// WorkflowStartEvent represents the start of a workflow, optionally nested.
type WorkflowStartEvent struct {
	Name           string `json:",omitempty"`
	OwnerURL       string `json:",omitempty"`
	ParentName     string `json:",omitempty"`
	ParentOwnerURL string `json:",omitempty"`
	SessionID      string `json:",omitempty"`
	SelectedTasks  string `json:",omitempty"`
	TagIDs         string `json:",omitempty"`
}

// NewWorkflowStartEvent creates a new WorkflowStartEvent.
func NewWorkflowStartEvent(name, ownerURL, parentName, parentOwnerURL, sessionID, selectedTasks, tagIDs string) *WorkflowStartEvent {
	return &WorkflowStartEvent{
		Name:           name,
		OwnerURL:       ownerURL,
		ParentName:     parentName,
		ParentOwnerURL: parentOwnerURL,
		SessionID:      sessionID,
		SelectedTasks:  selectedTasks,
		TagIDs:         tagIDs,
	}
}

// WorkflowEndEvent represents completion of a workflow.
type WorkflowEndEvent struct {
	Name           string `json:",omitempty"`
	OwnerURL       string `json:",omitempty"`
	ParentName     string `json:",omitempty"`
	ParentOwnerURL string `json:",omitempty"`
	SessionID      string `json:",omitempty"`
	Status         string `json:",omitempty"`
	Error          string `json:",omitempty"`
}

// NewWorkflowEndEvent creates a new WorkflowEndEvent.
func NewWorkflowEndEvent(name, ownerURL, parentName, parentOwnerURL, sessionID, status, errMsg string) *WorkflowEndEvent {
	return &WorkflowEndEvent{
		Name:           name,
		OwnerURL:       ownerURL,
		ParentName:     parentName,
		ParentOwnerURL: parentOwnerURL,
		SessionID:      sessionID,
		Status:         status,
		Error:          errMsg,
	}
}
