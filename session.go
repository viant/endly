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
	Username         string
	SuperUSerAuth    bool
	Path             *SystemPath
	EnvVariables     map[string]string
	CurrentDirectory string
	Deployed         map[string]string
	Cacheable        map[string]interface{}
	Mutex            *sync.RWMutex
}

//NewSystemTerminalSession create a new client session
func NewSystemTerminalSession(id string, connection ssh.Service) (*SystemTerminalSession, error) {
	return &SystemTerminalSession{
		ID:           id,
		Service:      connection,
		EnvVariables: make(map[string]string),
		Deployed:     make(map[string]string),
		Cacheable:    make(map[string]interface{}),
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
