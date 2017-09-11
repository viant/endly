package endly

import (
	"fmt"
)

type SessionInfo struct {
	Logs []*SessionLog
}

func (d *SessionInfo) Log(logEntry interface{}) error {
	switch entry := logEntry.(type) {

	case *CommandInfo:
		d.Logs = append(d.Logs, &SessionLog{Command: entry})
	case *TransferRequest:
		d.Logs = append(d.Logs, &SessionLog{Transfer: entry})
	default:
		return fmt.Errorf("Unsupported log entry: %T", logEntry)
	}
	return nil
}

func NewSessionInfo() *SessionInfo {
	return &SessionInfo{
		Logs: make([]*SessionLog, 0),
	}
}

type SessionLog struct {
	Command  *CommandInfo
	Transfer *TransferRequest
}
