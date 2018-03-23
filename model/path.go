package model

import "strings"

//Path represents a system path
type Path struct {
	index map[string]bool
	Items []string
}

//Push appends path to the system paths
func (p *Path) Push(paths ...string) {
	for _, path := range paths {
		if strings.Contains(path, "\n") {
			continue
		}
		if _, has := p.index[path]; has {
			continue
		}
		p.Items = append(p.Items, path)
		p.index[path] = true
	}
}

//EnvValue returns evn values
func (p *Path) EnvValue() string {
	return strings.Join(p.Items, ":")
}

//NewSystemPath create a new system path.
func NewPath(items ...string) *Path {
	return &Path{
		index: make(map[string]bool),
		Items: items,
	}
}
