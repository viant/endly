package webdriver

import "encoding/json"

// sessionCDPCommands retains the session's native or ChromeDriver transport.
// It lets ordinary call-driver commands share the same CDP implementation used
// by capture and native text insertion.
type sessionCDPCommands struct{ session *Session }

func (c sessionCDPCommands) CDPCommand(method string, params map[string]interface{}) (json.RawMessage, error) {
	return executeSessionCDP(c.session, method, params)
}
