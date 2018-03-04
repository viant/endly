package log

import "fmt"

type logRecordIterator struct {
	logFileProvider func() []*File
	logFiles        []*File
	logFileIndex    int
}

//HasNext returns true if iterator has next element.
func (i *logRecordIterator) HasNext() bool {
	var logFileCount = len(i.logFiles)
	if i.logFileIndex >= logFileCount {
		i.logFiles = i.logFileProvider()
		for j, candidate := range i.logFiles {
			if candidate.HasPendingLogs() {
				i.logFileIndex = j
				return true
			}
		}
		return false
	}

	logFile := i.logFiles[i.logFileIndex]
	if !logFile.HasPendingLogs() {
		i.logFileIndex++
		return i.HasNext()
	}
	return true
}



//Next sets item pointer with next element.
func (i *logRecordIterator) Next(itemPointer interface{}) error {
	var indexRecordPointer, ok = itemPointer.(*IndexedRecord)
	if ok {
		logFile := i.logFiles[i.logFileIndex]
		logRecord := logFile.ShiftLogRecordByIndex(indexRecordPointer.IndexValue)
		indexRecordPointer.Record = logRecord
		return nil
	}

	logRecordPointer, ok := itemPointer.(**Record)
	if !ok {
		return fmt.Errorf("expected *%T buy had %T", &Record{}, itemPointer)
	}
	logFile := i.logFiles[i.logFileIndex]
	logRecord := logFile.ShiftLogRecord()
	*logRecordPointer = logRecord
	return nil
}
