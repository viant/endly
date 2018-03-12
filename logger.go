package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
)

var logger = &Logger{
	Packages: make(map[string]bool),
}

//LogF global log function
var LogF = logger.Logf

//LogEnable enable log for supplied pacakge
var LogEnable = logger.Enable

//logger represents logger
type Logger struct {
	Packages map[string]bool
}

func (l *Logger) Enable(pkgs ...string) {
	for _, pkg := range pkgs {
		l.Packages[pkg] = true
	}
}

func (l *Logger) hasPackage(caller string) bool {
	if len(l.Packages) == 0 {
		return false
	}
	for candidate := range l.Packages {
		if strings.Contains(caller, candidate) {
			return true
		}
	}
	return false
}

//Logf logs message
func (l *Logger) Logf(template string, args ...interface{}) {
	if len(l.Packages) == 0 {
		return
	}
	caller, _, _ := toolbox.DiscoverCaller(2, 10, "logger.go")
	if !l.hasPackage(caller) {
		return
	}
	if !strings.Contains(template, "\n") {
		template += "\n"
	}
	fmt.Printf(template, args...)

}
