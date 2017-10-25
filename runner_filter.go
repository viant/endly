package endly


//RunnerReportingFilter runner reporting fiter
type RunnerReportingFilter struct {
	Stdin                   bool //log stdin
	Stdout                  bool //log stdout
	Transfer                bool //log transfer
	Task                    bool
	UseCase                 bool
	Action                  bool
	Deployment              bool
	SQLScript               bool
	PopulateDatastore       bool
	Sequence                bool
	RegisterDatastore       bool
	Assert                  bool
	DataMapping             bool
	HTTPTrip                bool
	Workflow                bool
	WorkflowParams          bool
	OnFailureFilter         *RunnerReportingFilter
	FirstUseCaseFailureOnly bool
}

//RunnerReportingOption represnets runner reporting options
type RunnerReportingOption struct {
	Filter *RunnerReportingFilter
}