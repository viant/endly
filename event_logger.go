package endly

import (
	"github.com/viant/toolbox"
	"fmt"
	"path"
	"os"
	"encoding/json"
	"strings"
)

type EventLogger struct {
	directory string
	subPath   string
	tagCount  map[string]int
	tagIndex  int
}



func (l *EventLogger) Log(event *Event) error {
	if event.Type == "Tag" {
		l.tagIndex++
		l.updateSubPath(event)
	}

	if _, has:= l.tagCount[l.subPath];!has {
		l.tagCount[l.subPath] = 0
	}
	l.tagCount[l.subPath]++

	var counter = l.tagCount[l.subPath]
	filename := path.Join(l.directory, l.subPath, fmt.Sprintf("%04d_%v.json", counter, event.Type))
	parent, _ := path.Split(filename)
	if ! toolbox.FileExists(parent) {
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



func (l *EventLogger) updateSubPath(event *Event) {
	if tag, ok := event.Value["tag"]; ok {
		l.subPath = strings.ToLower(fmt.Sprintf("%03d_%v_%v", l.tagIndex, event.Value["name"], tag))
	}
}


func NewEventLogger(directory string) *EventLogger {
	return &EventLogger{
		directory: directory,
		tagCount:  make(map[string]int),
		subPath:   "000_main",
	}
}
