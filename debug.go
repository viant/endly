package endly

import (
	"fmt"
	"github.com/viant/endly/common"
)

type Debug struct {
	Logs []*DebugLog
}

func (d *Debug) Log(logEntry interface{}) error {
	switch entry := logEntry.(type) {

	case *CommandInfo:
		d.Logs = append(d.Logs, &DebugLog{Command: entry})
	case *TransferRequest:
		d.Logs = append(d.Logs, &DebugLog{Transfer: entry})
	default:
		return fmt.Errorf("Unsupported log entry: %T", logEntry)
	}
	return nil
}

func NewDebug() *Debug {
	return &Debug{
		Logs: make([]*DebugLog, 0),
	}
}

type DebugLog struct {
	Command  *CommandInfo
	Transfer *TransferRequest
}

type DebugTransfer struct {
	SourceURL string
	TargetURL string
	Parsable  string
	Data      common.Map
}
