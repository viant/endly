package endly

import "github.com/viant/toolbox/url"

//NetworkTunnel represents network link, both local and remove needs to be in [host]:[port] format
type NetworkTunnel struct {
	Local  string `required:"true" description:"local [host]:[port]"`
	Remote string `required:"true" description:"remote [host]:[port]" `
}

//NetworkTunnelRequest represents SSH tunnel request
type NetworkTunnelRequest struct {
	Target  *url.Resource
	Tunnels []*NetworkTunnel
}

//NetworkTunnelResponse represents expanded net tunnel rule
type NetworkTunnelResponse struct {
	Forwards []*NetworkTunnel
}
