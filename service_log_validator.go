package endly

import (
	"bytes"
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	//LogValidatorServiceID represents log validator service id.
	LogValidatorServiceID = "validator/log"

	//LogValidatorServiceListenAction represents log listening action
	LogValidatorServiceListenAction = "listen"

	//LogValidatorServiceAssertAction represents log verification action
	LogValidatorServiceAssertAction = "assert"

	//LogValidatorServiceResetAction represents verification pending logs reset action
	LogValidatorServiceResetAction = "reset"
)

type logValidatorService struct {
	*AbstractService
}

//LogRecordAssert represents log record assert
type LogRecordAssert struct {
	TagID    string
	Expected interface{}
	Actual   interface{}
}

//LogAssertEvent represents log assert event
type LogAssertEvent struct {
	Type string
	Logs []*LogRecordAssert
}

//LogProcessingState represents log processing state
type LogProcessingState struct {
	Line     int
	Position int
}

//Update updates processed position and line number
func (s *LogProcessingState) Update(position, lineNumber int) (string, int) {
	s.Line = lineNumber
	s.Position += position
	return "", 0
}

//Reset resets processing state
func (s *LogProcessingState) Reset() {
	s.Line = 0
	s.Position = 0
}

//LogRecord repesents a log record
type LogRecord struct {
	URL    string
	Number int
	Line   string
}

//AsMap returns log records as map
func (r *LogRecord) AsMap() (map[string]interface{}, error) {
	var result = make(map[string]interface{})
	err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(r.Line)).Decode(&result)
	return result, err
}

//LogFile represents a log file
type LogFile struct {
	URL             string
	Content         string
	Name            string
	Exclusion       string
	Inclusion       string
	ProcessingState *LogProcessingState
	LastModified    time.Time
	Size            int
	Records         []*LogRecord
	Mutex           *sync.RWMutex
}

//ShiftLogRecord returns and remove the first log record if present
func (f *LogFile) ShiftLogRecord() *LogRecord {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	if len(f.Records) == 0 {
		return nil
	}
	result := f.Records[0]
	f.Records = f.Records[1:]
	return result
}

//PushLogRecord appends provided log record to the records.
func (f *LogFile) PushLogRecord(record *LogRecord) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	if len(f.Records) == 0 {
		f.Records = make([]*LogRecord, 0)
	}
	f.Records = append(f.Records, record)
}

//Reset resets processing state
func (f *LogFile) Reset(object storage.Object) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	f.Size = int(object.FileInfo().Size())
	f.LastModified = object.FileInfo().ModTime()
	f.ProcessingState.Reset()
}

//HasPendingLogs returns true if file has pending validation records
func (f *LogFile) HasPendingLogs() bool {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	return len(f.Records) > 0
}

func (f *LogFile) readLogRecords(reader io.Reader) error {
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
			f.PushLogRecord(&LogRecord{
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

//LogTypeMeta represents a log type meta
type LogTypeMeta struct {
	Source   *url.Resource
	LogType  *LogType
	LogFiles map[string]*LogFile
}

type logRecordIterator struct {
	logFileProvider func() []*LogFile
	logFiles        []*LogFile
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

func (s *logValidatorService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	switch actualRequest := request.(type) {
	case *LogValidatorListenRequest:
		response.Response, err = s.listen(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to run logValidator: %v, %v", actualRequest.Source, err)
		}
	case *LogValidatorAssertRequest:
		response.Response, err = s.assert(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to run logValidator: %v, %v", actualRequest, err)
		}
	case *LogValidatorResetRequest:
		response.Response, err = s.reset(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to run logValidator: %v, %v", actualRequest, err)
		}

	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *logValidatorService) reset(context *Context, request *LogValidatorResetRequest) (*LogValidatorResetResponse, error) {
	var response = &LogValidatorResetResponse{
		LogFiles: make([]string, 0),
	}
	for _, logTypeName := range request.LogTypes {
		if !s.state.Has(logTypeMetaKey(logTypeName)) {
			continue
		}
		if logTypeMeta, ok := s.state.Get(logTypeMetaKey(logTypeName)).(*LogTypeMeta); ok {
			for _, logFile := range logTypeMeta.LogFiles {
				logFile.ProcessingState = &LogProcessingState{
					Position: logFile.Size,
					Line:     len(logFile.Records),
				}
				logFile.Records = make([]*LogRecord, 0)
				response.LogFiles = append(response.LogFiles, logFile.Name)
			}
		}
	}
	return response, nil
}

func (s *logValidatorService) assert(context *Context, request *LogValidatorAssertRequest) (*LogValidatorAssertResponse, error) {

	var response = &LogValidatorAssertResponse{
		Description:    request.Description,
		ValidationInfo: make([]*ValidationInfo, 0),
	}
	var state = s.State()
	validator := &Validator{
		ExcludedFields: make(map[string]bool),
	}
	if len(request.ExpectedLogRecords) == 0 {
		return response, nil
	}

	if request.LogWaitTimeMs == 0 {
		request.LogWaitTimeMs = 500
	}
	if request.LogWaitRetryCount == 0 {
		request.LogWaitRetryCount = 3
	}

	var event = &LogAssertEvent{
		Logs: make([]*LogRecordAssert, 0),
	}

	for _, expectedLogRecords := range request.ExpectedLogRecords {
		logTypeMeta, err := s.getLogTypeMeta(expectedLogRecords, state)
		if err != nil {
			return nil, err
		}
		event.Type = expectedLogRecords.Type

		var logRecordIterator = logTypeMeta.LogRecordIterator()
		logWaitRetryCount := request.LogWaitRetryCount
		logWaitDuration := time.Duration(request.LogWaitTimeMs) * time.Millisecond

		for _, expectedLogRecord := range expectedLogRecords.Records {
			var validationInfo = &ValidationInfo{
				TagID:       expectedLogRecords.TagID,
				Description: fmt.Sprintf("Log Validation: %v", expectedLogRecords.Type),
			}
			response.ValidationInfo = append(response.ValidationInfo, validationInfo)
			for j := 0; j < logWaitRetryCount; j++ {
				if logRecordIterator.HasNext() {
					break
				}
				var sleepEventType = &SleepEventType{SleepTimeMs: int(logWaitDuration) / int(time.Millisecond)}
				AddEvent(context, sleepEventType, Pairs("value", sleepEventType))
				time.Sleep(logWaitDuration)
			}

			if !logRecordIterator.HasNext() {
				validationInfo.AddFailure(NewFailedTest(fmt.Sprintf("[%v]", expectedLogRecords.TagID), "Missing log record", expectedLogRecord, nil))
				return response, nil
			}

			var logRecord = &LogRecord{}
			logRecordIterator.Next(&logRecord)
			_, filename := toolbox.URLSplit(logRecord.URL)
			var actualLogRecord, err = logRecord.AsMap()
			if err != nil {
				return nil, err
			}

			event.Logs = append(event.Logs, &LogRecordAssert{
				TagID:    expectedLogRecords.TagID,
				Expected: expectedLogRecord,
				Actual:   actualLogRecord,
			})

			err = validator.Assert(expectedLogRecord, actualLogRecord, validationInfo, fmt.Sprintf("[%v:%v]", filename, logRecord.Number))
			if err != nil {
				return nil, err
			}
		}
		AddEvent(context, event, Pairs("value", event))
	}
	return response, nil
}

func (s *logValidatorService) getLogTypeMeta(expectedLogRecords *ExpectedLogRecord, state data.Map) (*LogTypeMeta, error) {
	var key = logTypeMetaKey(expectedLogRecords.Type)
	s.Mutex().Lock()
	defer s.Mutex().Unlock()
	if !state.Has(key) {
		return nil, fmt.Errorf("failed to assert, unknown type:%v, please call listen function with requested log type", expectedLogRecords.Type)
	}
	logTypeMeta := state.Get(key).(*LogTypeMeta)
	return logTypeMeta, nil
}

func (s *logValidatorService) readLogFile(context *Context, source *url.Resource, service storage.Service, candidate storage.Object, logType *LogType) (*LogTypeMeta, error) {
	var result *LogTypeMeta
	var key = logTypeMetaKey(logType.Name)
	s.Mutex().Lock()

	if !s.state.Has(logTypeMetaKey(logType.Name)) {
		s.state.Put(key, NewLogTypeMeta(source, logType))
	}
	if logTypeMeta, ok := s.state.Get(key).(*LogTypeMeta); ok {
		result = logTypeMeta
	}

	var isNewLogFile = false

	_, name := toolbox.URLSplit(candidate.URL())
	logFile, has := result.LogFiles[name]
	fileInfo := candidate.FileInfo()
	if !has {
		isNewLogFile = true

		logFile = &LogFile{
			Name:            name,
			Exclusion:       logType.Exclusion,
			Inclusion:       logType.Inclusion,
			URL:             candidate.URL(),
			LastModified:    fileInfo.ModTime(),
			Size:            int(fileInfo.Size()),
			ProcessingState: &LogProcessingState{},
			Mutex:           &sync.RWMutex{},
			Records:         make([]*LogRecord, 0),
		}
		result.LogFiles[name] = logFile
	}
	s.Mutex().Unlock()
	if !isNewLogFile && (logFile.Size == int(fileInfo.Size()) && logFile.LastModified.Unix() == fileInfo.ModTime().Unix()) {
		return result, nil
	}
	reader, err := service.Download(candidate)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
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

func (s *logValidatorService) readLogFiles(context *Context, service storage.Service, source *url.Resource, logTypes ...*LogType) (LogTypesMeta, error) {
	var err error
	source, err = context.ExpandResource(source)
	if err != nil {
		return nil, err
	}

	var response LogTypesMeta = make(map[string]*LogTypeMeta)
	candidates, err := service.List(source.URL)
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

func (s *logValidatorService) getStorageService(context *Context, resource *url.Resource) (storage.Service, error) {
	var state = context.state
	if state.Has(UseMemoryService) {
		return storage.NewMemoryService(), nil
	}
	return storage.NewServiceForURL(resource.URL, resource.Credential)
}

func (s *logValidatorService) listenForChanges(context *Context, request *LogValidatorListenRequest) error {
	var target, err = context.ExpandResource(request.Source)
	if err != nil {
		return err
	}
	service, err := s.getStorageService(context, target)
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
			_, err := s.readLogFiles(context, service, request.Source, request.Types...)
			if err != nil {
				log.Printf("failed to load log types %v", err)
				break
			}
			time.Sleep(frequency)
		}

	}()
	return nil
}

func (s *logValidatorService) listen(context *Context, request *LogValidatorListenRequest) (*LogValidatorListenResponse, error) {
	var source, err = context.ExpandResource(request.Source)
	if err != nil {
		return nil, err
	}

	for _, logType := range request.Types {
		if s.state.Has(logTypeMetaKey(logType.Name)) {
			return nil, fmt.Errorf("listener has been already register for %v", logType.Name)
		}
	}
	service, err := storage.NewServiceForURL(request.Source.URL, request.Source.Credential)
	if err != nil {
		return nil, err
	}
	defer service.Close()
	logTypeMetas, err := s.readLogFiles(context, service, request.Source, request.Types...)
	if err != nil {
		return nil, err
	}
	for _, logType := range request.Types {
		logMeta, ok := logTypeMetas[logType.Name]
		if !ok {
			logMeta = NewLogTypeMeta(source, logType)
			logTypeMetas[logType.Name] = logMeta
		}
		s.state.Put(logTypeMetaKey(logType.Name), logMeta)
	}

	response := &LogValidatorListenResponse{
		Meta: logTypeMetas,
	}

	err = s.listenForChanges(context, request)
	return response, err
}

//NewRequest creates a new request for provided action, (listen, asset,reset)
func (s *logValidatorService) NewRequest(action string) (interface{}, error) {
	switch action {
	case LogValidatorServiceListenAction:
		return &LogValidatorListenRequest{}, nil
	case LogValidatorServiceAssertAction:
		return &LogValidatorAssertRequest{}, nil
	case LogValidatorServiceResetAction:
		return &LogValidatorResetRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewLogValidatorService creates a new log validator service.
func NewLogValidatorService() Service {
	var result = &logValidatorService{
		AbstractService: NewAbstractService(LogValidatorServiceID,
			LogValidatorServiceListenAction,
			LogValidatorServiceAssertAction,
			LogValidatorServiceResetAction,
		),
	}
	result.AbstractService.Service = result
	return result
}

func logTypeMetaKey(name string) string {
	return fmt.Sprintf("meta_%v", name)
}

//Next sets item pointer with next element.
func (i *logRecordIterator) Next(itemPointer interface{}) error {
	var logRecordPointer, ok = itemPointer.(**LogRecord)
	if !ok {
		return fmt.Errorf("expected *%T buy had %T", &LogRecord{}, itemPointer)
	}
	logFile := i.logFiles[i.logFileIndex]
	logRecord := logFile.ShiftLogRecord()
	*logRecordPointer = logRecord
	return nil
}

//LogRecordIterator returns log record iterator
func (m *LogTypeMeta) LogRecordIterator() toolbox.Iterator {

	logFileProvider := func() []*LogFile {
		var result = make([]*LogFile, 0)
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

//NewLogTypeMeta creates a nre log type meta.
func NewLogTypeMeta(source *url.Resource, logType *LogType) *LogTypeMeta {
	return &LogTypeMeta{
		Source:   source,
		LogType:  logType,
		LogFiles: make(map[string]*LogFile),
	}
}
