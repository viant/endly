package endly

//RunnerReportingFilter runner reporting fiter
type RunnerReportingFilter struct {
	Stdin                   bool //log stdin
	Stdout                  bool //log stdout
	Transfer                bool //log transfer
	Deployment              bool
	Checkout                bool
	Build                   bool
	RegisterDatastore       bool
	SQLScript               bool
	Sequence                bool
	PopulateDatastore       bool
	Assert                  bool
	DataMapping             bool
	HTTPTrip                bool
	OnFailureFilter         *RunnerReportingFilter
	FirstUseCaseFailureOnly bool
}

//RunnerReportingOption represnets runner reporting options
type RunnerReportingOption struct {
	Filter *RunnerReportingFilter
}
