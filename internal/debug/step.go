package debug

type Step struct {
	Workflow string `json:"workflow,omitempty"`
	TaskName string `json:"task,omitempty"`
	Action   string `json:"action,omitempty"`
	TagID    string `json:"tagId,omitempty"`
	Kind     string `json:"kind,omitempty"`
}
