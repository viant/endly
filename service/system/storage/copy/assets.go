package copy

import (
	"github.com/viant/endly/model/location"
)

// Assets represents transfer assets
type Assets map[string]string

// AsTransfer converts map to transfer or transfers
func (t Assets) AsTransfer(base *Rule) []*Rule {
	var sourceBase, destBase = base.Source, base.Dest
	var transfers = make([]*Rule, 0)
	var isSourceRootPath = sourceBase != nil && sourceBase.URL != "" && sourceBase.Path() == "/"
	var isDestRootPath = destBase != nil && destBase.Path() == "/"
	for source, dest := range t {
		if dest == "" {
			dest = source
		}
		if isSourceRootPath {
			source = location.NewResource(source).Path()
		}
		if isDestRootPath {
			dest = location.NewResource(dest).Path()
		}
		transfer := &Rule{
			Source:       location.NewResource(source),
			Dest:         location.NewResource(dest),
			Substitution: base.Substitution,
			Compress:     base.Compress,
		}
		if sourceBase != nil {
			transfer.Source = JoinIfNeeded(sourceBase, source)

		}
		if destBase != nil {
			transfer.Dest = JoinIfNeeded(destBase, dest)
		}
		transfers = append(transfers, transfer)
	}
	return transfers
}
