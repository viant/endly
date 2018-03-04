package endly

import (
	"encoding/json"
	"fmt"
	"github.com/viant/toolbox"
	"log"
	"os"
	"path"
	"sync"
)

//EventLogger represent event logger to drop event details in the provied directory.
type EventLogger struct {
	listener         EventListener
	activities       *Activities
	directory        string
	workflowTag      string
	workflowTagCount int
	subPath          string
	tagCount         map[string]int
	mutex            *sync.Mutex
}

func (l *EventLogger) processEvent(event *Event) {
	if event.Value == nil {
		return
	}
	switch value := event.Value.(type) {
	case *Activity:
		l.activities.Push(value)
		l.updateSubpath()
	case *ActivityEndEvent:
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
	if l.workflowTag != l.activities.Last().TagID {
		l.workflowTagCount++
		l.workflowTag = l.activities.Last().TagID
		l.subPath = fmt.Sprintf("%03d_%v", l.workflowTagCount, l.activities.Last().TagID)
	}
}

func (l *EventLogger) handlerError(err error) {
	log.Print(err)
}

func (l *EventLogger) OnEvent(event *Event) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.processEvent(event)
	if _, has := l.tagCount[l.subPath]; !has {
		l.tagCount[l.subPath] = 0
	}
	l.tagCount[l.subPath]++
	var counter = l.tagCount[l.subPath]
	filename := path.Join(l.directory, l.subPath, fmt.Sprintf("%04d_%v.json", counter, event.Type()))
	parent, _ := path.Split(filename)
	if !toolbox.FileExists(parent) {
		err := os.MkdirAll(parent, 0744)
		if err != nil {
			l.handlerError(err)
			return
		}
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		l.handlerError(err)
		return
	}
	defer func() { _ = file.Close() }()

	buf, err := json.MarshalIndent(event.Value, "", "\t")
	if err != nil {
		l.handlerError(err)
		return
	}
	_, _ = file.Write(buf)
}

//Log logs an event
func (l *EventLogger) AsEventListener() EventListener {
	return func(event *Event) {
		if l.listener != nil {
			l.listener(event)
		}
		l.OnEvent(event)
	}
}

//NewEventLogger creates a new event logger
func NewEventLogger(directory string, listener EventListener) *EventLogger {
	var activities Activities = make([]*Activity, 0)
	var result = &EventLogger{
		listener:   listener,
		mutex:      &sync.Mutex{},
		directory:  directory,
		activities: &activities,
		tagCount:   make(map[string]int),
		subPath:    "000_main",
	}

	return result
}
