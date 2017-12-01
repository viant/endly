package endly

import (
	"github.com/viant/toolbox/ssh"
	"sync"
)

//SystemTerminalSession represents a system terminal session
type SystemTerminalSession struct {
	ID string
	ssh.MultiCommandSession
	DaemonType       int
	Service          ssh.Service
	OperatingSystem  *OperatingSystem
	envVariables     map[string]string
	currentDirectory string
	Deployed         map[string]string
	Sdk              map[string]*SystemSdkInfo
	Mutex            *sync.RWMutex
}

//NewSystemTerminalSession create a new client session
func NewSystemTerminalSession(id string, connection ssh.Service) (*SystemTerminalSession, error) {
	return &SystemTerminalSession{
		ID:           id,
		Service:      connection,
		envVariables: make(map[string]string),
		Deployed:     make(map[string]string),
		Sdk:          make(map[string]*SystemSdkInfo),
		Mutex:        &sync.RWMutex{},
	}, nil
}



//SystemTerminalSessions represents a map of client sessions keyed by session id
type SystemTerminalSessions map[string]*SystemTerminalSession

//Has checks if client session exists for provided id.
func (s *SystemTerminalSessions) Has(id string) bool {
	_, has := (*s)[id]
	return has
}

var systemTerminalSessionsKey = (*SystemTerminalSessions)(nil)
