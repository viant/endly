package exec

import (
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
)

//TerminalSession returns SystemTerminalSession for passed in target resource.
func TerminalSession(context *endly.Context, target *url.Resource) (*endly.SystemTerminalSession, error) {
	sessions := context.TerminalSessions()
	if target == nil {
		return nil, errors.New("target was empty")
	}
	if !sessions.Has(target.Host()) {
		service, err := context.Service(ServiceID)
		if err != nil {
			return nil, err
		}
		response := service.Run(context, &OpenSessionRequest{
			Target: target,
		})
		if response.Err != nil {
			return nil, response.Err
		}
	}
	return sessions[target.Host()], nil
}
