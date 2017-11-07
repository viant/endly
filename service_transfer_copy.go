package endly

import (
	"fmt"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
)

//TransferCopyRequest represents a resources copy request
type TransferCopyRequest struct {
	Transfers []*Transfer // transfers
}

//TransferCopyResponse represents a resources copy response
type TransferCopyResponse struct {
	Transferred []*TransferLog //transferred logs
}

//Transfer represents copy instruction
type Transfer struct {
	Source   *url.Resource     //source URL with credential
	Target   *url.Resource     //target URL with credential
	Expand   bool              //flag to substitute content with state keys
	Compress bool              //flag to compress asset before sending over wirte and to decompress (this option is only supported on scp or file proto)
	Replace  map[string]string //replacements map, if key if found in the conent it wil be replaced with corresponding value.
}

//TransferLog represents transfer log
type TransferLog struct {
	SourceURL   string
	TargetURL   string
	Error       string
	Substituted string
	State       data.Map
}

//NewTransferLog create a new transfer log
func NewTransferLog(context *Context, source, target string, err error, expand bool) *TransferLog {
	result := &TransferLog{
		SourceURL: source,
		TargetURL: target,
	}
	if expand {
		var state = context.State()
		result.State = state.AsEncodableMap()
	}
	if err != nil {
		result.Error = fmt.Sprintf("%v", err)
	}
	return result
}
