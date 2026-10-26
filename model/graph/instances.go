package graph

import (
	"fmt"
	"github.com/viant/afs/storage"
	"github.com/viant/afs/url"
	"github.com/viant/toolbox"
	"math"
	"regexp"
	"strings"
)

type (
	Instances struct {
		BaseURL   string
		Instances []*Instance
		ByIdx     map[int]int
		Min       int
		Max       int
		Match     string
		expr      *regexp.Regexp
	}

	Instance struct {
		Object storage.Object
		Tag    string
		Match  string
		Index  int
	}
)

func (i *Instances) Range() string {
	if i == nil {
		return ""
	}
	max := i.Lookup(i.Max)
	if max == nil {
		return ""
	}
	return fmt.Sprintf("%v..%v", i.Min, max.Match)
}

func (i *Instances) Lookup(index int) *Instance {
	if i == nil {
		return nil
	}
	pos, ok := i.ByIdx[index]
	if !ok || pos < 0 || pos >= len(i.Instances) {
		return nil
	}
	return i.Instances[pos]
}

func NewInstances(holderURL string, match string, objects []storage.Object) *Instances {
	result, _ := NewInstancesChecked(holderURL, match, objects)
	return result
}

// NewInstancesChecked reports invalid template patterns instead of panicking.
func NewInstancesChecked(holderURL string, match string, objects []storage.Object) (*Instances, error) {
	result := &Instances{
		ByIdx: map[int]int{},
		Min:   math.MaxInt,
	}
	matchExpr := strings.Replace(match, "${index}", "(\\d+)", 1)
	matchExpr = strings.Replace(matchExpr, "*", ".+", 1)
	expr, err := regexp.Compile(matchExpr)
	if err != nil {
		return result, fmt.Errorf("invalid template instance pattern %q: %w", match, err)
	}
	for i, object := range objects {
		if url.Equals(object.URL(), holderURL) {
			continue
		}
		name := object.Name()
		groups := expr.FindAllStringSubmatch(name, 1)
		if len(groups) == 0 || len(groups[0]) <= 1 {
			continue
		}
		pos := len(result.Instances)
		instance := &Instance{
			Object: objects[i],
			Match:  groups[0][1],
		}
		instance.Index = toolbox.AsInt(strings.TrimLeft(instance.Match, "0"))
		instance.Tag = strings.Trim(strings.Replace(name, instance.Match, "", 1), "_")

		result.Instances = append(result.Instances, instance)
		result.ByIdx[instance.Index] = pos
		if instance.Index > result.Max {
			result.Max = instance.Index
		}
		if instance.Index < result.Min || result.Min == 0 {
			result.Min = instance.Index
		}
	}
	return result, nil
}
