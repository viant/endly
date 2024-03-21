package transfer

type (

	// Project represents a project
	Project struct {
		//ID represents project ID
		ID string
		//Name represents project name
		Name string
		//Description represents project description
		Description string
	}

	// Workflow represents a workflow
	Workflow struct {
		//ID represents workflow ID
		ID string
		//
		Revision string
		//URI represents workflow
		URI string
		//ProjectID represents project ID
		ProjectID string
		//Name represents workflow name
		Name string
		//	Description represents workflow description
		Init []string `jsonx:"inline"`
		//	Description represents workflow description
		Post []string `jsonx:"inline"`
		//	Description represents workflow description
		Steps []*Task
		//Template represents workflow template
		Template string
	}

	// Task represents a workflow step
	Task struct {
		//ID represents step ID
		ID string
		//ParentId represents parent step ID
		ParentId string
		//Index represents step index within parent
		Index int `sqlx:"IDX"`
		//Tag represents step tag
		Tag string
		//TagIndex represent index within template
		TagIndex int
		//Init represents step init variables
		Init []string `jsonx:"inline"`
		//Post represents step post variables
		Post []string `jsonx:"inline"`
		//Description represents step description
		Description string
		//When represents step when expression
		When string `sqlx:"WHEN_EXPR"`
		//Exit represents step exit expression
		Exit string
		//OnError represents task to continue on error
		OnError string
		//Deferred represents step deffered expression
		Deferred string

		//action attributes
		//Service represents action service
		Service string
		//Action represents action name
		Action string
		//Request represents action request
		Request string
		//RequestURI represents action request reference
		RequestURI string
		//Async represents action async flag
		Async bool
		//Skip       represents action skip flag
		Skip string
		//Template task
		Template bool
		//SubPath template subpath
		SubPath string
		//Range represents template range
		Range string
		//Data represents template data
		Data map[string]string `jsonx:"inline"`

		//repeater attributes
		//Variables represents repeater variables
		Variables []string `jsonx:"inline"`
		//Extracts represents repeater extracts
		Extracts Extracts `jsonx:"inline"`
		//SleepTimeMs represents repeater sleep time in milliseconds
		SleepTimeMs int
		//ThinkTimeMs represents repeater think time in milliseconds
		ThinkTimeMs int
		//Logging represents repeater logging flag
		Logging *bool
		//Repeat represents repeater repeat count
		Repeat int
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

func (t *Task) SetID(prefix, name string) {
	t.ID = prefix + "/" + name
	t.Tag = name
}
