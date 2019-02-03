package registry


//ContractAdapter  represents a contract adapter
type ContractAdapter interface {
	SetService(service interface{}) error
	Call() (result interface{}, err error)
	GetId() string
}


var registry = make(map[string]ContractAdapter, 0)

func Register(adapter ContractAdapter) {
	registry[adapter.GetId()] = adapter
}

func Get(id string) (ContractAdapter, bool) {
	result, ok := registry[id]
	return result, ok
}