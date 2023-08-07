package shared

// ContractAdapter  represents a contract adapter
type ContractAdapter interface {
	SetService(service interface{}) error
	Call() (result interface{}, err error)
	GetId() string
}
