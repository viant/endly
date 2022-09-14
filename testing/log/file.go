package log

import (
	"fmt"
	"github.com/viant/afs/storage"
	"github.com/viant/endly"
	"github.com/viant/endly/model/msg"
	"github.com/viant/endly/workflow"
	"github.com/viant/toolbox"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"time"
)

//File represents a log file
type File struct {
	URL     string
	Content string
	Name    string
	*Type
	ProcessingState *ProcessingState
	LastModified    time.Time
	Size            int
	Records         []*Record
	IndexedRecords  map[string]*Record
	Mutex           *sync.RWMutex
	context         *endly.Context
}

//ShiftLogRecord returns and remove the first log record if present
func (f *File) ShiftLogRecord() *Record {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	if len(f.Records) == 0 {
		return nil
	}
	result := f.Records[0]
	f.Records = f.Records[1:]
	if f.Type.Debug {
		info, _ := toolbox.AsJSONText(result)
		_ = endly.Run(f.context, &workflow.PrintRequest{
			Style:   msg.MessageStyleOutput,
			Message: fmt.Sprintf("shift [%v] -> %v", f.Type.Name, info),
		}, nil)
	}
	return result
}

//ShiftLogRecordByIndex returns and remove the first log record if present
func (f *File) ShiftLogRecordByIndex(value string) (*Record, bool) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	if len(f.Records) == 0 {
		return nil, false
	}
	result, has := f.IndexedRecords[value]
	if !has {
		result = f.Records[0]
		f.Records = f.Records[1:]
	} else {
		var records = make([]*Record, 0)
		for _, candidate := range f.Records {
			if candidate == result {
				continue
			}
			records = append(records, candidate)
		}
		f.Records = records
	}

	if f.Type.Debug {
		info, _ := toolbox.AsJSONText(result)
		_ = endly.Run(f.context, &workflow.PrintRequest{
			Style:   msg.MessageStyleOutput,
			Message: fmt.Sprintf("shifted [%v:idx:%s]-> %v", f.Type.Name, value, info),
		}, nil)
	}

	return result, has
}

//PushLogRecord appends provided log record to the records.
func (f *File) PushLogRecord(record *Record) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	if len(f.Records) == 0 {
		f.Records = make([]*Record, 0)
	}

	indexValue := ""
	f.Records = append(f.Records, record)
	if f.UseIndex() {
		if expr, err := f.GetIndexExpr(); err == nil {
			indexValue = matchLogIndex(expr, record.Line)
			if indexValue != "" {
				f.IndexedRecords[indexValue] = record
			}
		}
	}
	if f.Type.Debug {
		if indexValue != "" {
			indexValue = " idx:" + indexValue
		}
		info, _ := toolbox.AsJSONText(record)
		_ = endly.Run(f.context, &workflow.PrintRequest{
			Style:   msg.MessageStyleInput,
			Message: fmt.Sprintf("push [%v%v] <- %v", f.Type.Name, indexValue, info),
		}, nil)
	}

}

//Reset resets processing state
func (f *File) Reset(object storage.Object) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	f.Size = int(object.Size())
	f.LastModified = object.ModTime()
	f.ProcessingState.Reset()
}

//HasPendingLogs returns true if file has pending validation records
func (f *File) HasPendingLogs() bool {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	return len(f.Records) > 0
}

func (f *File) readLogRecords(reader io.Reader) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	if f.ProcessingState.Position > len(data) {
		return nil
	}
	var line = ""
	var startPosition = f.ProcessingState.Position
	var startLine = f.ProcessingState.Line
	var lineIndex = startLine
	var dataProcessed = 0
	for i := startPosition; i < len(data); i++ {
		dataProcessed++
		aChar := string(data[i])
		if aChar != "\n" && aChar != "\r" {
			line += aChar
			continue
		}

		line = strings.Trim(line, " \r\t")
		lineIndex++
		if f.Exclusion != "" {
			if strings.Contains(line, f.Exclusion) {
				line, dataProcessed = f.ProcessingState.Update(dataProcessed, lineIndex)
				continue
			}
		}
		if f.Inclusion != "" {
			if !strings.Contains(line, f.Inclusion) {
				line, dataProcessed = f.ProcessingState.Update(dataProcessed, lineIndex)
				continue
			}
		}

		if len(line) > 0 {
			f.PushLogRecord(&Record{
				URL:    f.URL,
				Line:   line,
				Number: lineIndex,
			})
		}
		if err != nil {
			return err
		}
		line, dataProcessed = f.ProcessingState.Update(dataProcessed, lineIndex)
	}
	return nil
}
