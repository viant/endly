package model

import (
	"github.com/viant/toolbox/ssh"
	"sync"
)

//Session represents a system terminal session
type Session struct {
	ID               string
	ssh.MultiCommandSession
	DaemonType       int
	Service          ssh.Service
	Os               *OperatingSystem
	Username         string
	SuperUSerAuth    bool
	Path             *Path
	EnvVariables     map[string]string
	CurrentDirectory string
	Deployed         map[string]string
	Cacheable        map[string]interface{}
	Mutex            *sync.RWMutex
}

//NewSession create a new client session
func NewSession(id string, connection ssh.Service) (*Session, error) {
	return &Session{
		ID:           id,
		Service:      connection,
		EnvVariables: make(map[string]string),
		Deployed:     make(map[string]string),
		Cacheable:    make(map[string]interface{}),
		Mutex:        &sync.RWMutex{},
	}, nil
}

//Sessions represents a map of client sessions keyed by session id
type Sessions map[string]*Session

//Has checks if client session exists for provided id.
func (s *Sessions) Has(id string) bool {
	_, has := (*s)[id]
	return has
}


