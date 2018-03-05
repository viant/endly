package cli

//Filter runner reporting fiter
type Filter struct {
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
	OnFailureFilter         *Filter
	Report                  map[string]bool
	FirstUseCaseFailureOnly bool
}

//RunnerReportingOptions represnets runner reporting options
type RunnerReportingOptions struct {
	Filter *Filter
}

//DefaultRunnerReportingOption returns new default reporting options
func DefaultRunnerReportingOption() *RunnerReportingOptions {
	return &RunnerReportingOptions{
		Filter: &Filter{
			Stdin:             true,
			Stdout:            true,
			Transfer:          true,
			SQLScript:         true,
			PopulateDatastore: true,
			Sequence:          true,
			RegisterDatastore: true,
			DataMapping:       true,
			OnFailureFilter: &Filter{
				HTTPTrip: true,
				Stdin:    true,
				Stdout:   true,
			},
		},
	}
}
