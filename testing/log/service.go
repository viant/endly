package log

import (
	"bytes"
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/criteria"
	estorage "github.com/viant/endly/system/storage"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/endly/udf"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	//ServiceID represents log validator service id.
	ServiceID = "validator/log"
)

type service struct {
	*endly.AbstractService
}

func (s *service) reset(context *endly.Context, request *ResetRequest) (*ResetResponse, error) {
	var response = &ResetResponse{
		LogFiles: make([]string, 0),
	}
	for _, logTypeName := range request.LogTypes {

		var state = s.State()
		if !state.Has(logTypeMetaKey(logTypeName)) {
			continue
		}
		if logTypeMeta, ok := state.Get(logTypeMetaKey(logTypeName)).(*TypeMeta); ok {
			for _, logFile := range logTypeMeta.LogFiles {
				logFile.ProcessingState = &ProcessingState{
					Position: logFile.Size,
					Line:     len(logFile.Records),
				}
				logFile.Records = make([]*Record, 0)
				response.LogFiles = append(response.LogFiles, logFile.Name)
			}
		}
	}
	return response, nil
}

func (s *service) assert(context *endly.Context, request *AssertRequest) (*AssertResponse, error) {
	var response = &AssertResponse{
		Validations: make([]*assertly.Validation, 0),
	}
	if len(request.Expect) == 0 {
		return response, nil
	}

	for _, expectedLogRecords := range request.Expect {
		typeMeta, err := s.getLogTypeMeta(expectedLogRecords)
		if err != nil {
			return nil, err
		}
		var recordIterator = typeMeta.Iterator()

		var aMap = data.NewMap()
		aMap.Put("logType", expectedLogRecords.Type)
		aMap.Put("TagID", expectedLogRecords.TagID)

		for _, expectedRecord := range expectedLogRecords.Records {
			var validation = &assertly.Validation{
				TagID:       expectedLogRecords.TagID,
				Description: aMap.ExpandAsText(request.DescriptionTemplate),
			}
			response.Validations = append(response.Validations, validation)
			if !s.waitForRecord(context, recordIterator, request) {
				validation.AddFailure(assertly.NewFailure("", fmt.Sprintf("[%v]", expectedLogRecords.TagID), "missing log record", expectedRecord, nil))
				return response, nil
			}

			var logRecord = &Record{}
			logRecord, err := s.matchLogRecord(typeMeta, expectedRecord, recordIterator)
			if err != nil {
				return nil, err
			}
			var actualLogRecord interface{} = logRecord.Line
			if isLogStructured := toolbox.IsMap(expectedRecord); isLogStructured {
				actualLogRecord, err = logRecord.AsMap()
				if err != nil {
					return nil, err
				}
			}
			logRecordsAssert := &validator.TaggedAssert{
				TagID:    expectedLogRecords.TagID,
				Expected: expectedRecord,
				Actual:   actualLogRecord,
			}
			_, filename := toolbox.URLSplit(logRecord.URL)
			logValidation, err := criteria.Assert(context, fmt.Sprintf("%v:%v", filename, logRecord.Number), expectedRecord, actualLogRecord)
			if err != nil {
				return nil, err
			}
			context.Publish(logRecordsAssert)
			context.Publish(logValidation)
			validation.MergeFrom(logValidation)
		}
	}
	return response, nil
}

func (s *service) waitForRecord(context *endly.Context, recordIterator toolbox.Iterator, request *AssertRequest) bool {
	for j := 0; j < request.LogWaitRetryCount; j++ {
		if recordIterator.HasNext() {
			return true
		}
		s.Sleep(context, int(request.LogWaitTimeMs)/int(time.Millisecond))
	}
	return recordIterator.HasNext()
}

func (s *service) matchLogRecord(typeMeta *TypeMeta, expectedRecord interface{}, logRecordIterator toolbox.Iterator) (*Record, error) {
	var calledNext = false
	var logRecord *Record
	if typeMeta.LogType.UseIndex() {
		if expr, err := typeMeta.LogType.GetIndexExpr(); err == nil {
			var expectedTextRecord = toolbox.AsString(expectedRecord)
			if toolbox.IsMap(expectedRecord) || toolbox.IsSlice(expectedRecord) || toolbox.IsStruct(expectedRecord) {
				expectedTextRecord, _ = toolbox.AsJSONText(expectedRecord)
			}
			var indexValue = matchLogIndex(expr, expectedTextRecord)
			if indexValue != "" {
				indexedLogRecord := &IndexedRecord{
					IndexValue: indexValue,
				}
				err = logRecordIterator.Next(indexedLogRecord)
				if err != nil {
					return nil, err
				}
				calledNext = true
				logRecord = indexedLogRecord.Record
			}
		}
	}

	if !calledNext {
		if err := logRecordIterator.Next(&logRecord); err != nil {
			return nil, err
		}
	}
	return logRecord, nil
}

func (s *service) getLogTypeMeta(expectedLogRecords *TypedRecord) (*TypeMeta, error) {
	var key = logTypeMetaKey(expectedLogRecords.Type)
	s.Mutex().Lock()
	defer s.Mutex().Unlock()
	var state = s.State()
	if !state.Has(key) {
		return nil, fmt.Errorf("failed to assert, unknown type:%v, please call listen function with requested log type", expectedLogRecords.Type)
	}
	logTypeMeta := state.Get(key).(*TypeMeta)
	return logTypeMeta, nil
}

//tryReadSnapshot tries to read file snapshot, since file may change any time, this method attempts to get a stable snapshot read withhout actual change in file content while it is read.
func (s *service) tryReadSnapshot(service storage.Service, object storage.Object, attemptsCount int) (io.Reader, error) {
	fileSize := object.FileInfo().Size()
	for i := 0; i < attemptsCount; i++ {
		reader, err := service.Download(object)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		content, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		if recentSnapshot, err := service.StorageObject(object.URL()); err == nil {
			if recentSnapshot.FileInfo().Size() == fileSize { //no content modification since last content download, it safe to return snapshot
				return bytes.NewReader(content), nil
			}
			fileSize = recentSnapshot.FileInfo().Size()
		}
		time.Sleep(300 * time.Millisecond)
	}
	return nil, nil
}

func (s *service) readLogFile(context *endly.Context, source *url.Resource, service storage.Service, candidate storage.Object, logType *Type) (*TypeMeta, error) {
	var result *TypeMeta
	var key = logTypeMetaKey(logType.Name)
	s.Mutex().Lock()

	var state = s.State()
	if !state.Has(key) {
		state.Put(key, NewTypeMeta(source, logType))
	}

	if logTypeMeta, ok := state.Get(key).(*TypeMeta); ok {
		result = logTypeMeta
	}

	var isNewLogFile = false
	_, name := toolbox.URLSplit(candidate.URL())
	logFile, has := result.LogFiles[name]
	fileInfo := candidate.FileInfo()
	if !has {
		isNewLogFile = true
		logFile = &File{
			context:         context,
			Type:            logType,
			Name:            name,
			URL:             candidate.URL(),
			LastModified:    fileInfo.ModTime(),
			Size:            int(fileInfo.Size()),
			ProcessingState: &ProcessingState{},
			Mutex:           &sync.RWMutex{},
			Records:         make([]*Record, 0),
			IndexedRecords:  make(map[string]*Record),
		}
		result.LogFiles[name] = logFile
	}
	s.Mutex().Unlock()

	if !isNewLogFile && (logFile.Size == int(fileInfo.Size()) && logFile.LastModified.Unix() == fileInfo.ModTime().Unix()) {
		return result, nil
	}

	reader, err := s.tryReadSnapshot(service, candidate, 3)
	if err != nil || reader == nil {
		return nil, err
	}
	if logFile.UDF != "" {
		content, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		transformed, err := udf.TransformWithUDF(context, logFile.UDF, logFile.UDF, content)
		switch payload := transformed.(type) {
		case string:
			reader = ioutil.NopCloser(strings.NewReader(payload))
		case []byte:
			reader = ioutil.NopCloser(bytes.NewReader(payload))
		default:
			return nil, fmt.Errorf("unsupported response type expeced string or []byte but had: %T", transformed)
		}
	}

	logContent, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var content = string(logContent)
	var fileOverridden = false
	if len(logFile.Content) > len(content) { //log shrink or rolled over case
		logFile.Reset(candidate)
		logFile.Content = content
		fileOverridden = true
	}

	if !fileOverridden && logFile.Size < int(fileInfo.Size()) && !strings.HasPrefix(content, string(logFile.Content)) {
		logFile.Reset(candidate)
	}

	logFile.Content = content
	logFile.Size = len(logContent)
	if len(logContent) > 0 {
		err = logFile.readLogRecords(bytes.NewReader(logContent))
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *service) readLogFiles(context *endly.Context, service storage.Service, source *url.Resource, logTypes ...*Type) (TypesMeta, error) {
	var err error
	source, err = context.ExpandResource(source)
	if err != nil {
		return nil, err
	}

	var response TypesMeta = make(map[string]*TypeMeta)
	candidates, err := service.List(source.URL)
	if err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		if candidate.IsFolder() {
			continue
		}
		for _, logType := range logTypes {
			mask := strings.Replace(logType.Mask, "*", ".+", len(logType.Mask))
			maskExpression, err := regexp.Compile("^" + mask + "$")
			if err != nil {
				return nil, err
			}
			_, name := toolbox.URLSplit(candidate.URL())
			if maskExpression.MatchString(name) {
				logTypeMeta, err := s.readLogFile(context, source, service, candidate, logType)
				if err != nil {
					return nil, err
				}
				response[logType.Name] = logTypeMeta
			}
		}
	}
	return response, nil
}

func (s *service) listenForChanges(context *endly.Context, request *ListenRequest) error {
	var target, err = context.ExpandResource(request.Source)
	if err != nil {
		return err
	}
	service, err := estorage.GetStorageService(context, target)
	if err != nil {
		return err
	}
	go func() {
		defer service.Close()
		frequency := time.Duration(request.FrequencyMs) * time.Millisecond
		if request.FrequencyMs <= 0 {
			frequency = 400 * time.Millisecond
		}
		for !context.IsClosed() {
			_, err := s.readLogFiles(context, service, target, request.Types...)
			if err != nil {
				log.Printf("failed to load log types %v", err)
				break
			}
			time.Sleep(frequency)
		}

	}()
	return nil
}

func (s *service) listen(context *endly.Context, request *ListenRequest) (*ListenResponse, error) {
	var source, err = context.ExpandResource(request.Source)
	if err != nil {
		return nil, err
	}
	var state = s.State()
	for _, logType := range request.Types {
		if state.Has(logTypeMetaKey(logType.Name)) {
			return nil, fmt.Errorf("listener has been already register for %v", logType.Name)
		}
	}
	service, err := storage.NewServiceForURL(source.URL, source.Credentials)
	if err != nil {
		return nil, err
	}
	defer service.Close()
	logTypeMetas, err := s.readLogFiles(context, service, source, request.Types...)
	if err != nil {
		return nil, err
	}
	for _, logType := range request.Types {
		logMeta, ok := logTypeMetas[logType.Name]
		if !ok {
			logMeta = NewTypeMeta(source, logType)
			logTypeMetas[logType.Name] = logMeta
		}
		state.Put(logTypeMetaKey(logType.Name), logMeta)
	}

	response := &ListenResponse{
		Meta: logTypeMetas,
	}
	err = s.listenForChanges(context, request)
	return response, err
}

const (
	logValidatorExample = `{
  "FrequencyMs": 500,
  "Source": {
    "URL": "scp://127.0.0.1/opt/elogger/logs/",
    "Credentials": "${env.HOME}/.secret/localhost.json"
  },
  "Types": [
    {
      "Name": "event1",
      "Format": "json",
      "Mask": "elog*.log",
      "Inclusion": "/event1/",
      "IndexRegExpr": "\"EventID\":\"([^\"]+)\""
    }
  ]
}`

	logValidatorAssertExample = ` {
		"LogWaitTimeMs": 5000,
		"LogWaitRetryCount": 5,
		"Description": "E-logger event log validation",
		"Expect": [
			{
				"Type": "event1",
				"Records": [
					{
						"EventID": "84423348-1384-11e8-b0b4-ba004c285304",
						"EventType": "event1",
						"Request": {
							"Method": "GET",
							"URL": "http://127.0.0.1:8777/event1/?k10=v1\u0026k2=v2"
						}
					},
					{
						"EventID": "8441c4bc-1384-11e8-b0b4-ba004c285304",
						"EventType": "event1",
						"Request": {
							"Method": "GET",
							"URL": "http://127.0.0.1:8777/event1/?k1=v1\u0026k2=v2"
						}
					}
				]
			},
			{
				"Type": "event2",
				"Records": [
					{
						"EventID": "84426d4a-1384-11e8-b0b4-ba004c285304",
						"EventType": "event2",
						"Request": {
							"Method": "GET",
							"URL": "http://127.0.0.1:8777/event2/?k1=v1\u0026k2=v2"
						}
					}
				]
			}
		]
	}`
)

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "listen",
		RequestInfo: &endly.ActionInfo{
			Description: "check for log changes",
			Examples: []*endly.UseCase{
				{
					Description: "log listen",
					Data:        logValidatorExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &ListenRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ListenResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ListenRequest); ok {
				return s.listen(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "assert",
		RequestInfo: &endly.ActionInfo{
			Description: "assert queued logs",
			Examples: []*endly.UseCase{
				{
					Description: "assert",
					Data:        logValidatorAssertExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &AssertRequest{}
		},
		ResponseProvider: func() interface{} {
			return &AssertResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*AssertRequest); ok {
				return s.assert(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "reset",
		RequestInfo: &endly.ActionInfo{
			Description: "reset logs queues",
		},
		RequestProvider: func() interface{} {
			return &ResetRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ResetResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ResetRequest); ok {
				return s.reset(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new log validator service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
