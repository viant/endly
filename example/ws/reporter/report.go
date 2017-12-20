package reporter

import "github.com/viant/dsc"

type Report interface {
	GetType() string

	GetName() string

	SQL(manager dsc.Manager, parameters map[string]interface{}) (string, error)

	Unwrap() interface{}
}

type Reports map[string]Report

type ReportProvider func(report interface{}) (Report, error)

type ReportProviders map[string]ReportProvider
