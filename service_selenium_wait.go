package endly



//SeleniumWait represents selenium wait data
type SeleniumWait struct {
	Repeat       int
	SleepInMs    int
	ExitCriteria string
}



func (r *SeleniumWait) Data() (int, int, string) {
	var repeat = 1
	var sleepInMs = 0
	var exitCriteria = ""
	if r != nil {
		if r.Repeat > 0 {
			repeat = r.Repeat
		}
		sleepInMs = r.SleepInMs
		exitCriteria = r.ExitCriteria
	}
	return repeat, sleepInMs, exitCriteria
}

