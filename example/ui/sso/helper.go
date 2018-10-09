package sso

import (
	"net/http"
	"strings"
)

func extractIp(request *http.Request) string {
	var ip = request.RemoteAddr
	xForwardedFor := request.Header.Get("X-Forwarded-For")
	if len(xForwardedFor) > 0 {
		if strings.Index(xForwardedFor, ",") != -1 {
			var ips = strings.Split(xForwardedFor, ",")
			return ips[len(ips)-1]
		}
		return xForwardedFor
	}

	portIndex := strings.LastIndex(ip, ":")
	if portIndex != -1 {
		ip = string(ip[:portIndex])
	}
	return ip
}
