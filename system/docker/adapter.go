package docker

import "golang.org/x/net/context"

// ContractAdapter  represents a contract adapter
type ContractAdapter interface {
	SetService(service interface{}) error
	SetContext(ctx context.Context)
	Call() (result interface{}, err error)
	GetId() string
}
