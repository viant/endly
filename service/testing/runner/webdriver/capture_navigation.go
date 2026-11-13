package webdriver

// Drain before the old document/target can discard its completed response bodies.
// This is an explicit capture boundary, even inside the periodic drain throttle.
// The driver remains untouched so backend-specific capabilities stay available.
func drainBeforeNavigation(session *Session, method string, parameters []interface{}) {
	if session == nil || session.Capture == nil {
		return
	}
	switch method {
	case "Get", "Refresh", "Back", "Forward", "SwitchWindow", "Close", "Quit", "OpenTab", "Tabs", "SwitchTab", "NewTab", "CloseTab":
	case "CDPCommand":
		if len(parameters) == 0 {
			return
		}
		command, _ := parameters[0].(string)
		switch command {
		case "Page.navigate", "Page.reload", "Page.navigateToHistoryEntry", "Target.closeTarget", "Target.activateTarget", "Target.createTarget":
		default:
			return
		}
	default:
		return
	}
	session.Capture.DrainNow(session)
}
