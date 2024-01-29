package kms

import (
	"google.golang.org/api/cloudkms/v1"
	"reflect"
	"sort"
)

// Policy represents kms policy
type Policy struct {
	Bindings []*cloudkms.Binding
	Version  int64
}

// ShallUpdatePolicy returns true if policy needs to be updated
func ShallUpdatePolicy(prev, policy *Policy) bool {
	if policy == nil {
		return false
	}
	if prev == nil {
		return true
	}

	if len(prev.Bindings) != len(policy.Bindings) {
		return true
	}
	destRoles := make(map[string][]string)
	indexBindings(destRoles, policy.Bindings)

	sourceRoles := make(map[string][]string)
	indexBindings(sourceRoles, prev.Bindings)

	for k, destMembers := range destRoles {
		sourceMembers, ok := sourceRoles[k]
		if !ok {
			return true
		}
		if !reflect.DeepEqual(sourceMembers, destMembers) {
			return true
		}
	}
	return false
}

func indexBindings(index map[string][]string, bindings []*cloudkms.Binding) {
	if len(bindings) == 0 {
		return
	}
	for i := range bindings {
		members := bindings[i].Members
		sort.Strings(members)
		index[bindings[i].Role] = members
	}
}
