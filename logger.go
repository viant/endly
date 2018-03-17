package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"path"
	"strings"
)

var logger = &Logger{
	Packages: make(map[string]bool),
}

//LogF global log function
var LogF = logger.Logf

//EnableLogging enable log for supplied package
var EnableLogging = logger.Enable

//Returns true if logging is enabled for supplied package
var IsLoggingEnabled = logger.IsEnabled

//logger represents logger
type Logger struct {
	Packages map[string]bool
}

func (l *Logger) Enable(pkgs ...string) {
	for _, pkg := range pkgs {
		l.Packages[pkg] = true
	}
}

func (l *Logger) IsEnabled(pkg string) bool {
	if len(l.Packages) == 0 {
		return false
	}
	for candidate := range l.Packages {
		if strings.Contains(pkg, candidate) {
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
	parent, _ := path.Split(caller)
	pkg, _ := path.Split(parent)
	if !l.IsEnabled(pkg) {
		return
	}
	if !strings.Contains(template, "\n") {
		template += "\n"
	}
	fmt.Printf(template, args...)

}
