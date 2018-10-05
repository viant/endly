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
	"strings"
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
	activityPath     string
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

func (l *Logger) normalizeActivityPath(activity *model.Activity) string {
	if activity == nil || activity.Action == "" {
		return ""
	}
	if activity.Action == "run" && activity.Service == ServiceID {
		req := toolbox.AsMap(activity.Request)
		if URL, ok := req["assetURL"]; ok {
			_, name := toolbox.URLSplit(toolbox.AsString(URL))
			return name
		}
	}
	var service = strings.Replace(activity.Service, "/", "_", 1)
	var action = strings.Replace(activity.Action, "-", "_", 1)
	return service + "_" + action + "/"
}

func (l *Logger) getAndIncrementTag(tag string) int {
	if _, has := l.tagCount[tag]; !has {
		l.tagCount[tag] = 0
	}
	l.tagCount[tag]++
	return l.tagCount[tag]
}

//OnEvent handles supplied event.
func (l *Logger) OnEvent(event msg.Event) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.processEvent(event)

	activity := l.Last()
	activityID := l.normalizeActivityPath(activity)

	if activityID != "" {
		if !strings.HasSuffix(l.activityPath, activityID) {
			activityCount := l.getAndIncrementTag(l.subPath + activityID)
			l.activityPath = fmt.Sprintf("%03d_%v", activityCount, activityID)
		}
		activityID = l.activityPath
	}

	subPath := l.subPath
	tagCount := l.getAndIncrementTag(l.subPath)

	if activityID != "" {
		tagCount = l.getAndIncrementTag(l.subPath + activityID)
		subPath = subPath + "/" + activityID
	}

	filename := path.Join(l.directory, subPath, fmt.Sprintf("%04d_%v.json", tagCount, event.Type()))
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
	value := event.Value()

	var aMap = map[string]interface{}{}
	if err := toolbox.DefaultConverter.AssignConverted(&aMap, value); err == nil {
		value = toolbox.DeleteEmptyKeys(aMap)
	}

	buf, err := json.MarshalIndent(value, "", "\t")
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
		Activities:   model.NewActivities(),
		Listener:     listener,
		mutex:        &sync.Mutex{},
		directory:    directory,
		tagCount:     make(map[string]int),
		subPath:      "000_main",
		activityPath: "",
	}

	return result
}
