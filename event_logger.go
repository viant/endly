package endly

import (
	"encoding/json"
	"fmt"
	"github.com/viant/toolbox"
	"os"
	"path"
	"reflect"
	"sync"
)

//EventLogger represent event logger to drop event details in the provied directory.
type EventLogger struct {
	activities *Activities
	directory  string
	workflowTag      string
	workflowTagCount int
	subPath          string
	tagCount map[string]int
	mutex *sync.Mutex
}

func (l *EventLogger) processEvent(event *Event) {

	var canidate = event.get(reflect.TypeOf(&WorkflowServiceActivity{}))
	if canidate != nil {
		activity, _ := canidate.(*WorkflowServiceActivity)
		l.activities.Push(activity)
		l.updateSubpath()
	}

	if event.Type == "WorkflowRunRequest.End" {
		if len(*l.activities) > 0 {
			l.activities.Pop()
			l.updateSubpath()
		}
	}
}
func (l *EventLogger) updateSubpath() {
	if len(*l.activities) == 0 {
		return
	}
	if l.workflowTag != l.activities.Last().TagId {
		l.workflowTagCount++
		l.workflowTag = l.activities.Last().TagId
		l.subPath = fmt.Sprintf("%03d_%v", l.workflowTagCount, l.activities.Last().TagId)
	}
}

//Log logs an event
func (l *EventLogger) Log(event *Event) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.processEvent(event)
	if _, has := l.tagCount[l.subPath]; !has {
		l.tagCount[l.subPath] = 0
	}
	l.tagCount[l.subPath]++

	var counter = l.tagCount[l.subPath]

	filename := path.Join(l.directory, l.subPath, fmt.Sprintf("%04d_%v.json", counter, event.Type))
	parent, _ := path.Split(filename)
	if !toolbox.FileExists(parent) {
		err := os.MkdirAll(parent, 0744)
		if err != nil {
			return err
		}
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	buf, err := json.MarshalIndent(event.Value, "", "\t")
	if err != nil {
		return fmt.Errorf("Failed to log %v, %v", event.Type, err)
	}
	_, err = file.Write(buf)
	return err
}

//NewEventLogger creates a new event logger
func NewEventLogger(directory string) *EventLogger {
	var activities Activities = make([]*WorkflowServiceActivity, 0)
	return &EventLogger{
		mutex:&sync.Mutex{},
		directory:  directory,
		activities: &activities,
		tagCount:   make(map[string]int),
		subPath:    "000_main",
	}
}
