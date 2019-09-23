package transfer

import (
	"github.com/viant/toolbox/url"

)

//Assets represents transfer assets
type Assets map[string]string


//AsTransfer converts map to transfer or transfers
func (t Assets) AsTransfer(base *Rule) []*Rule {
	var sourceBase, destBase = base.Source, base.Dest

	var transfers = make([]*Rule, 0)
	var isSourceRootPath = sourceBase != nil && sourceBase.ParsedURL != nil && sourceBase.ParsedURL.Path == "/"
	var isDestRootPath = destBase != nil && destBase.ParsedURL != nil && destBase.ParsedURL.Path == "/"
	for source, dest := range t {
		if dest == "" {
			dest = source
		}
		if isSourceRootPath {
			source = url.NewResource(source).ParsedURL.Path
		}
		if isDestRootPath {
			dest = url.NewResource(dest).ParsedURL.Path
		}
		transfer := &Rule{
			Source:   url.NewResource(source),
			Dest:     url.NewResource(dest),
			Substitution:base.Substitution,
			Compress: base.Compress,
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

