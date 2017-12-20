package reporter

import "github.com/viant/dsc"

//Report represents a generic report
type Report interface {
	GetType() string

	GetName() string

	SQL(manager dsc.Manager, parameters map[string]interface{}) (string, error)

	Unwrap() interface{}
}

//Reports represents map of reports
type Reports map[string]Report

//ReportProvider represents a report provider
type ReportProvider func(report interface{}) (Report, error)

//ReportProviders represents a report providers
type ReportProviders map[string]ReportProvider
