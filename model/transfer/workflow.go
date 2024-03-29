package transfer

import "encoding/json"

type (
	Project struct {
		ID          string `json:"ID,omitempty"`
		Name        string `json:"NAME,omitempty"`
		Description string `json:"DESCRIPTION,omitempty"`
	}
	Workflow struct {
		ID            string  `json:"ID,omitempty"`
		Position      int     `json:"POSITION,omitempty"`
		ParentId      string  `json:"PARENT_ID,omitempty"`
		Revision      string  `json:"REVISION,omitempty"`
		URI           string  `json:"URI,omitempty"`
		ProjectID     string  `json:"PROJECT_ID,omitempty"`
		Name          string  `json:"NAME,omitempty"`
		Init          string  `jsonx:"inline" json:"INIT,omitempty"`
		Post          string  `jsonx:"inline" json:"POST,omitempty"`
		Steps         []*Task `json:"-"`
		Template      string  `json:"TEMPLATE,omitempty"`
		InstanceIndex int     `json:"INSTANCE_INDEX,omitempty"`
		InstanceTag   string  `json:"INSTANCE_TAG,omitempty"`
	}

	Task struct {
		ID          string   `json:"ID,omitempty"`
		WorkflowID  string   `json:"WORKFLOW_ID,omitempty"`
		ParentId    string   `json:"PARENT_ID,omitempty"`
		Position    int      `json:"POSITION,omitempty"`
		Tag         string   `json:"TAG,omitempty"`
		Init        string   `jsonx:"inline" json:"INIT,omitempty"`
		Post        string   `jsonx:"inline" json:"POST,omitempty"`
		Description string   `json:"DESCRIPTION,omitempty"`
		When        string   `sqlx:"WHEN_EXPR" json:"WHEN_EXPR,omitempty"`
		Exit        string   `sqlx:"EXIT_EXPR" json:"EXIT_EXPR,omitempty"`
		OnError     string   `json:"ON_ERROR,omitempty"`
		Deferred    string   `json:"DEFERRED,omitempty"`
		Service     string   `json:"SERVICE,omitempty"`
		Action      string   `json:"ACTION,omitempty"`
		Input       string   `json:"INPUT,omitempty"`
		InputURI    string   `json:"INPUT_URI,omitempty"`
		Async       bool     `json:"ASYNC,omitempty"`
		Skip        string   `sqlx:"SKIP_EXPR" json:"SKIP_EXPR,omitempty"`
		Fail        bool     `json:"FAIL,omitempty"`
		IsTemplate  bool     `json:"IS_TEMPLATE,omitempty"`
		SubPath     string   `json:"SUB_PATH,omitempty"`
		Range       string   `sqlx:"RANGE_EXPR" json:"RANGE_EXPR,omitempty"`
		Data        string   `jsonx:"inline" json:"DATA,omitempty"`
		Variables   string   `jsonx:"inline" json:"VARIABLES,omitempty"`
		Extracts    Extracts `jsonx:"inline" json:"EXTRACTS,omitempty"`
		SleepTimeMs int      `json:"SLEEP_TIME_MS,omitempty"`
		ThinkTimeMs int      `json:"THINK_TIME_MS,omitempty"`
		Logging     *bool    `json:"LOGGING,omitempty"`
		Repeat      int      `sqlx:"REPEAT_RUN" json:"REPEAT_RUN,omitempty"`
		InstanceIndex int     `json:"INSTANCE_INDEX,omitempty"`
		InstanceTag   string  `json:"INSTANCE_TAG,omitempty"`
	}
	//Revision represents a workflow revision
	Revision struct {
		//ID represents revision ID
		ID string
		//WorkflowID represents workflow ID
		Principal string
		//Comment represents revision comment
		Comment string
		//Diff represents revision diff
		Diff string
	}

	Extract struct {
		RegExpr  string `description:"regular expression with oval bracket to extract match pattern"`            //regular expression
		Key      string `description:"state key to store a match"`                                               //state key to store a match
		Reset    bool   `description:"reset the key in the context before evaluating this data extraction rule"` //reset the key in the context before evaluating this data extraction rule
		Required bool   `description:"require that at least one pattern match is returned"`                      //require that at least one pattern match is returned
	}

	Extracts []*Extract
)

func (t *Task) GetData() map[string]string {
	data := make(map[string]string)
	if err := json.Unmarshal([]byte(t.Data), &data); err != nil {
		return nil
	}
	return data
}

func (t *Task) SetID(prefix, name string) {
	t.ID = prefix + "/" + name
	t.Tag = name
}
