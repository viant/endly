package endly

import (
	"github.com/viant/toolbox/ssh"
	"github.com/viant/toolbox/url"
)

//OpenSessionRequest represents an open session request.
type OpenSessionRequest struct {
	Target          *url.Resource      //Session is created from target host (servername, port)
	Config          *ssh.SessionConfig //ssh configuration
	SystemPaths     []string           //system path that are applied to the ssh session
	Env             map[string]string
	Transient       bool   //if this flag is true, caller is responsible for closing session, othewise session is closed as context is closed
	CommandsBasedir string //capture all ssh service command in supplied dir (for unit test only)
	ReplayService   ssh.Service //use Ssh ReplayService instead of actual SSH service (for unit test only)
}

//OpenSessionResponse represents a session id
type OpenSessionResponse struct {
	SessionID string
}

//CloseSessionRequest closes session
type CloseSessionRequest struct {
	SessionID string
}

//CloseSessionResponse closes session response
type CloseSessionResponse struct {
	SessionID string
}
