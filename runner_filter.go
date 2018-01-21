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

//RunnerReportingOptions represnets runner reporting options
type RunnerReportingOptions struct {
	Filter *RunnerReportingFilter
}

//DefaultRunnerReportingOption returns new default reporting options
func DefaultRunnerReportingOption() *RunnerReportingOptions {
	return &RunnerReportingOptions{
		Filter: &RunnerReportingFilter{
			Stdin:             true,
			Stdout:            true,
			Transfer:          true,
			SQLScript:         true,
			PopulateDatastore: true,
			Sequence:          true,
			RegisterDatastore: true,
			DataMapping:       true,
			OnFailureFilter: &RunnerReportingFilter{
				HTTPTrip: true,
				Stdin:    true,
				Stdout:   true,
			},
		},
	}
}
