package endly

import "github.com/viant/toolbox/url"

//NetworkTunnel represents network link
type NetworkTunnel struct {
	Local  string
	Remote string
}

//NetworkRequest represents a net forwarding request
type NetworkTunnelRequest struct {
	Target  *url.Resource
	Tunnels []*NetworkTunnel
}

//NetworkTunnelResponse represents expanded net tunnel rule
type NetworkTunnelResponse struct {
	Forwards []*NetworkTunnel
}
