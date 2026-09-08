package mobile

type Check struct {
	Name        string
	Status      string
	Path        string
	Version     string
	Detail      string
	Remediation string
}

func ChecksReady(checks []Check, required map[string]bool) bool {
	for _, check := range checks {
		if required[check.Name] && check.Status != "ok" {
			return false
		}
	}
	return true
}
