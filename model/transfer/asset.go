package transfer

// Asset represents a workflow asset
type Asset struct {
	ID            string `json:"ID,omitempty"`
	Location      string `json:"LOCATION,omitempty"`
	Description   string `json:"DESCRIPTION,omitempty"`
	WorkflowID    string `json:"WORKFLOW_ID,omitempty"`
	IsDir         bool   `json:"IS_DIR,omitempty"`
	Template      string `json:"TEMPLATE,omitempty"`
	InstanceIndex int    `json:"INSTANCE_INDEX,omitempty"`
	InstanceTag   string `json:"INSTANCE_TAG,omitempty"`
	Position      int    `json:"POSITION,omitempty"`
	Source        []byte `json:"SOURCE,omitempty"`
	Format        string `json:"FORMAT,omitempty"`
	Codec         string `json:"CODEC,omitempty"`
}