package network

import "github.com/viant/toolbox/url"

//NetworkTunnel represents network link, both local and remove needs to be in [host]:[port] format
type NetworkTunnel struct {
	Local  string `required:"true" description:"local [host]:[port]"`
	Remote string `required:"true" description:"remote [host]:[port]" `
}

//TunnelRequest represents SSH tunnel request
type TunnelRequest struct {
	Target  *url.Resource
	Tunnels []*NetworkTunnel
}

//TunnelResponse represents expanded net tunnel rule
type TunnelResponse struct {
	Forwards []*NetworkTunnel
}
