package markdown

import (
	"sort"
	"strings"
)

type (
	assets struct {
		workflows []string
		info      map[string]*info
	}

	info struct {
		loaded   bool
		uploaded bool
	}
)

func (a *assets) lookup(URL string) *info {
	return a.info[URL]
}

func (a *assets) shallUpload(URL string) bool {
	_, ok := a.info[URL]
	if ok {
		return false
	}
	if strings.HasSuffix(URL, ".yaml") {
		a.workflows = append(a.workflows, URL)
	}
	a.info[URL] = &info{uploaded: true}
	return true
}

func (a *assets) rootWorkflowURL() string {
	for _, candidate := range a.workflows {
		if strings.HasPrefix(candidate, "run.yaml") {
			return candidate
		}
	}
	sort.Slice(a.workflows, func(i, j int) bool {
		if strings.Count(a.workflows[i], "/") < strings.Count(a.workflows[j], "/") {
			return true
		}
		return len(a.workflows[i]) < len(a.workflows[j])
	})
	return a.workflows[0]
}

func newAssets() *assets {
	return &assets{info: make(map[string]*info)}
}
