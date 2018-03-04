package log

import (
	"github.com/viant/toolbox/url"
	"github.com/viant/toolbox"
	"sort"
)

//TypesMeta represents log type meta details
type TypesMeta map[string]*TypeMeta


//TypeMeta represents a log type meta
type TypeMeta struct {
	Source   *url.Resource
	LogType  *Type
	LogFiles map[string]*File
}



//Iterator returns log record iterator
func (m *TypeMeta) Iterator() toolbox.Iterator {
	logFileProvider := func() []*File {
		var result = make([]*File, 0)
		for _, logFile := range m.LogFiles {
			result = append(result, logFile)
		}
		sort.Slice(result, func(i, j int) bool {
			var left = result[i].LastModified
			var right = result[j].LastModified
			if !left.After(right) && !right.After(left) {
				return result[i].URL > result[j].URL
			}
			return left.After(right)
		})
		return result
	}

	return &logRecordIterator{
		logFiles:        logFileProvider(),
		logFileProvider: logFileProvider,
	}
}

//NewTypeMeta creates a nre log type meta.
func NewTypeMeta(source *url.Resource, logType *Type) *TypeMeta {
	return &TypeMeta{
		Source:   source,
		LogType:  logType,
		LogFiles: make(map[string]*File),
	}
}
