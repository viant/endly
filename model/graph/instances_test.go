package graph

import (
	"strings"
	"testing"

	"github.com/viant/afs/storage"
)

func TestInstancesInvalidStateDoesNotPanic(t *testing.T) {
	cases := map[string]*Instances{
		"nil":             nil,
		"empty":           {},
		"missing maximum": {Min: 1, Max: 3, ByIdx: map[int]int{1: 0}, Instances: []*Instance{{Index: 1, Match: "001"}}},
		"invalid offset":  {Min: 1, Max: 3, ByIdx: map[int]int{3: 2}, Instances: []*Instance{{Index: 1, Match: "001"}}},
		"negative offset": {Min: 1, Max: 3, ByIdx: map[int]int{3: -1}},
	}
	for name, instances := range cases {
		t.Run(name, func(t *testing.T) {
			if got := instances.Range(); got != "" {
				t.Fatalf("invalid range = %q", got)
			}
			if got := instances.Lookup(3); got != nil {
				t.Fatalf("invalid lookup = %#v", got)
			}
		})
	}
}

type instanceObject struct {
	storage.Object
	name     string
	location string
}

func (o instanceObject) Name() string { return o.name }
func (o instanceObject) URL() string  { return o.location }

func TestInstancesCheckedPreservesIndexedDiscovery(t *testing.T) {
	objects := []storage.Object{
		instanceObject{name: "cases", location: "file:///cases"},
		instanceObject{name: "001_one", location: "file:///cases/001_one"},
		instanceObject{name: "003_three", location: "file:///cases/003_three"},
		instanceObject{name: "notes", location: "file:///cases/notes"},
	}
	instances, err := NewInstancesChecked("file:///cases", "${index}_*", objects)
	if err != nil {
		t.Fatal(err)
	}
	if got := instances.Range(); got != "1..003" {
		t.Fatalf("range = %q", got)
	}
	if len(instances.Instances) != 2 || instances.Lookup(1).Match != "001" || instances.Lookup(3).Tag != "three" || instances.Lookup(2) != nil {
		t.Fatalf("discovery changed: %#v", instances)
	}
	legacy := NewInstances("file:///cases", "${index}_*", objects)
	if legacy.Range() != instances.Range() {
		t.Fatal("legacy constructor changed valid range")
	}
}

func TestInstancesCheckedInvalidPattern(t *testing.T) {
	instances, err := NewInstancesChecked("file:///cases", "[${index}", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid template instance pattern") {
		t.Fatalf("error = %v", err)
	}
	if instances.Range() != "" {
		t.Fatal("invalid pattern yielded instances")
	}
	if NewInstances("file:///cases", "[${index}", nil).Range() != "" {
		t.Fatal("legacy invalid pattern yielded instances")
	}
}
