package workflow

import (
	"encoding/json"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/viant/endly/model"
	"github.com/viant/endly/model/msg"
	"github.com/viant/toolbox"
)

// Logger represent event logger to drop event details in the provied directory.
type Logger struct {
	*model.Activities
	Listener  msg.Listener
	directory string
	// hierarchical stacks and sequencing
	workflowStack    []string
	taskStack        []string
	templateTagStack []*model.MetaTag
	groupSeq         map[string]int
	groupSeqCounter  int
	fileSeq          map[string]int
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
	case *model.ActivityEndEvent:
		l.activityEnded = true
	case *WorkflowStartEvent:
		if value != nil {
			seg := l.deriveWorkflowSegment(value)
			l.workflowStack = append(l.workflowStack, seg)
		}
	case *WorkflowEndEvent:
		if len(l.workflowStack) > 0 {
			l.workflowStack = l.workflowStack[:len(l.workflowStack)-1]
		}
	case *model.TaskStartEvent:
		if value != nil {
			l.taskStack = append(l.taskStack, value.TaskName)
			if value.TemplateTag != nil {
				tagCopy := *value.TemplateTag
				l.templateTagStack = append(l.templateTagStack, &tagCopy)
			} else {
				l.templateTagStack = append(l.templateTagStack, nil)
			}
		}
	case *model.TaskEndEvent:
		if len(l.taskStack) > 0 {
			l.taskStack = l.taskStack[:len(l.taskStack)-1]
		}
		if len(l.templateTagStack) > 0 {
			l.templateTagStack = l.templateTagStack[:len(l.templateTagStack)-1]
		}
	}
}

func normalizeSegment(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return strings.ToLower(s)
}

func (l *Logger) groupKey() string {
	wf := strings.Join(l.workflowStack, "/")
	ts := strings.Join(l.taskStack, "/")
	return wf + "|" + ts
}

func (l *Logger) groupPath() string {
	parts := make([]string, 0, len(l.workflowStack)+len(l.taskStack))
	for _, w := range l.workflowStack {
		n := normalizeSegment(w)
		if n != "" {
			parts = append(parts, n)
		}
	}
	for i, t := range l.taskStack {
		n := normalizeSegment(t)
		if n == "" {
			continue
		}
		// include template tag name for this task level if present
		if i < len(l.templateTagStack) {
			if tag := l.templateTagStack[i]; tag != nil && tag.Tag != "" {
				n = n + "_" + normalizeSegment(tag.Tag)
			}
		}
		parts = append(parts, n)
	}
	if len(parts) == 0 {
		parts = append(parts, "main")
	}
	return strings.Join(parts, "_")
}

func (l *Logger) groupDir() string {
	key := l.groupKey()
	if l.groupSeq == nil {
		l.groupSeq = map[string]int{}
	}
	if _, ok := l.groupSeq[key]; !ok {
		l.groupSeqCounter++
		l.groupSeq[key] = l.groupSeqCounter
	}
	seq := l.groupSeq[key]
	return sprintf4(seq) + "_" + l.groupPath()
}

func sprintf4(n int) string {
	s := "0000" + toolbox.AsString(n)
	return s[len(s)-4:]
}

func (l *Logger) nextFileSeq(groupDir string) int {
	if l.fileSeq == nil {
		l.fileSeq = map[string]int{}
	}
	l.fileSeq[groupDir] = l.fileSeq[groupDir] + 1
	return l.fileSeq[groupDir]
}

func (l *Logger) leafFileName(event msg.Event, groupDir string) string {
	seq := sprintf4(l.nextFileSeq(groupDir))
	t := event.Type()
	lowerType := strings.ToLower(t)
	isRequest := strings.HasSuffix(lowerType, "request")
	isResponse := strings.HasSuffix(lowerType, "response") || event.Init() != nil

	if isRequest || isResponse {
		service := ""
		action := ""
		if l.Activities != nil && l.Activities.Last() != nil {
			service = normalizeSegment(l.Activities.Last().Service)
			action = normalizeSegment(l.Activities.Last().Action)
		}
		if service == "" || action == "" {
			base := t
			if isRequest && strings.HasSuffix(t, "Request") {
				base = t[:len(t)-len("Request")]
			} else if strings.HasSuffix(t, "Response") {
				base = t[:len(t)-len("Response")]
			}
			fragments := strings.Split(base, "_")
			if len(fragments) >= 2 {
				service = normalizeSegment(fragments[0])
				action = normalizeSegment(strings.Join(fragments[1:], "_"))
			} else {
				service = normalizeSegment(base)
				action = ""
			}
		}
		serviceAction := service
		if action != "" {
			serviceAction += "_" + action
		}
		suffix := "request"
		if isResponse {
			suffix = "response"
		}
		return seq + "_" + serviceAction + "_" + suffix + ".json"
	}
	return seq + "_" + strings.ToLower(t) + ".json"
}

func (l *Logger) shouldWriteEvent(event msg.Event) bool {
	if event == nil || event.Value() == nil {
		return false
	}
	switch event.Value().(type) {
	case *model.TaskStartEvent, *model.TaskEndEvent:
		return false
	default:
		return true
	}
}

func (l *Logger) deriveWorkflowSegment(e *WorkflowStartEvent) string {
	// Prefer explicit workflow name if meaningful
	name := normalizeSegment(e.Name)
	if name != "" && name != "run" {
		return name
	}
	// Fallback to owner URL parent segment, skipping common folders like "cases"
	base := e.OwnerURL
	if base == "" {
		return name
	}
	// Normalize and extract parent segments
	u := strings.TrimSuffix(base, "/")
	// Strip scheme if present
	if idx := strings.Index(u, "://"); idx != -1 {
		u = u[idx+3:]
	}
	// Remove file name if any
	if i := strings.LastIndex(u, "/"); i != -1 {
		parent := u[:i]
		last := parent
		if j := strings.LastIndex(parent, "/"); j != -1 {
			last = parent[j+1:]
			// if immediate parent is "cases", pick grandparent
			if strings.ToLower(last) == "cases" {
				grand := parent[:j]
				if k := strings.LastIndex(grand, "/"); k != -1 {
					last = grand[k+1:]
				} else if len(grand) > 0 {
					last = grand
				}
			}
		}
		seg := normalizeSegment(last)
		if seg != "" {
			return seg
		}
	}
	return name
}

func (l *Logger) handlerError(err error) {
	log.Print(err)
}

// legacy helpers removed in hierarchical logger

// OnEvent handles supplied event.
func (l *Logger) OnEvent(event msg.Event) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.processEvent(event)

	// Skip writing certain structural events
	if !l.shouldWriteEvent(event) {
		return
	}

	groupDir := l.groupDir()
	filename := path.Join(l.directory, groupDir, l.leafFileName(event, groupDir))
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
	var payload interface{} = value
	if value != nil {
		var aMap = map[string]interface{}{}
		if err := toolbox.DefaultConverter.AssignConverted(&aMap, value); err == nil {
			payload = toolbox.DeleteEmptyKeys(aMap)
		}
	}

	workflowName := ""
	if len(l.workflowStack) > 0 {
		workflowName = l.workflowStack[len(l.workflowStack)-1]
	}
	taskPath := make([]string, len(l.taskStack))
	copy(taskPath, l.taskStack)
	var templateTag *model.MetaTag
	if n := len(l.templateTagStack); n > 0 {
		templateTag = l.templateTagStack[n-1]
	}
	var tagName string
	if templateTag != nil {
		tagName = templateTag.Tag
	}
	wrapper := map[string]interface{}{
		"type":        strings.ToLower(event.Type()),
		"workflow":    workflowName,
		"taskPath":    taskPath,
		"tag":         tagName,
		"templateTag": templateTag,
		"event":       payload,
	}

	buf, err := json.MarshalIndent(wrapper, "", "\t")
	if err != nil {
		l.handlerError(err)
		return
	}
	_, _ = file.Write(buf)
}

// AsEventListener returns an event Listener
func (l *Logger) AsEventListener() msg.Listener {
	return func(event msg.Event) {
		if l.Listener != nil {
			l.Listener(event)
		}
		l.OnEvent(event)
	}
}

// New creates a new event logger
func NewLogger(directory string, listener msg.Listener) *Logger {
	var result = &Logger{
		Activities:    model.NewActivities(),
		Listener:      listener,
		mutex:         &sync.Mutex{},
		directory:     directory,
		workflowStack: []string{},
		taskStack:     []string{},
		groupSeq:      map[string]int{},
		fileSeq:       map[string]int{},
	}

	return result
}
