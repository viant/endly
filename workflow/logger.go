package workflow

import (
	"encoding/json"
	"fmt"
	"github.com/viant/endly/model"
	"github.com/viant/endly/msg"
	"github.com/viant/toolbox"
	"log"
	"os"
	"path"
	"sync"
)

//Logger represent event logger to drop event details in the provied directory.
type Logger struct {
	*model.Activities
	Listener         msg.Listener
	directory        string
	workflowTag      string
	workflowTagCount int
	subPath          string
	tagCount         map[string]int
	mutex            *sync.Mutex
	activityEnded    bool
}

func (l *Logger) processEvent(event msg.Event) {
	if event.Value() == nil {
		return
	}
	switch value := event.Value().(type) {
	case *model.Activity:

		if l.activityEnded && l.Len() > 0 {
			l.activityEnded = false
			l.Pop()
		}

		l.Push(value)
		l.updateSubpath()
	case *model.ActivityEndEvent:
		l.activityEnded = true

	}
}

func (l *Logger) updateSubpath() {
	if l.Len() == 0 {
		return
	}
	tagID := l.Last().TagID
	if l.workflowTag != tagID {
		l.workflowTagCount++
		l.workflowTag = tagID
		l.subPath = fmt.Sprintf("%03d_%v", l.workflowTagCount, tagID)
	}
}

func (l *Logger) handlerError(err error) {
	log.Print(err)
}

//OnEvent handles supplied event.
func (l *Logger) OnEvent(event msg.Event) {
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

	buf, err := json.MarshalIndent(event.Value(), "", "\t")
	if err != nil {
		l.handlerError(err)
		return
	}
	_, _ = file.Write(buf)
}

//AsEventListener returns an event Listener
func (l *Logger) AsEventListener() msg.Listener {
	return func(event msg.Event) {
		if l.Listener != nil {
			l.Listener(event)
		}
		l.OnEvent(event)
	}
}

//New creates a new event logger
func NewLogger(directory string, listener msg.Listener) *Logger {
	var result = &Logger{
		Activities: model.NewActivities(),
		Listener:   listener,
		mutex:      &sync.Mutex{},
		directory:  directory,
		tagCount:   make(map[string]int),
		subPath:    "000_main",
	}

	return result
}
