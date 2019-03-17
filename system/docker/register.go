package docker

var registry = make(map[string]ContractAdapter, 0)

//Register register an adapter
func register(adapter ContractAdapter) {
	id := adapter.GetId()
	registry[id] = adapter
}
