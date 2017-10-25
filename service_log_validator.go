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

//LogValidatorServiceID represents log validator service id.
const LogValidatorServiceID = "validator/log"

type logValidatorService struct {
	*AbstractService
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
	SkipExpression  string
	ProcessingState *LogProcessingState
	LastModified    *time.Time
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
	var len = len(record.Line)
	if len > 40 {
		len = 40
	}
	f.Records = append(f.Records, record)
}

//Reset resets processing state
func (f *LogFile) Reset(object storage.Object) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	f.Size = int(object.Size())
	f.LastModified = object.LastModified()
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
		if f.SkipExpression != "" {
			if strings.Contains(line, f.SkipExpression) {
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
	logFileProvider func(blacklist ...*LogFile) []*LogFile
	logFiles        []*LogFile
	logFileIndex    int
}

//HasNext returns true if iterator has next element.
func (i *logRecordIterator) HasNext() bool {
	if i.logFileIndex >= len(i.logFiles) {
		i.logFiles = i.logFileProvider(i.logFiles...)
		i.logFileIndex = 0
		if i.logFileIndex < len(i.logFiles) {
			return true
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
			response.Error = fmt.Sprintf("Failed to run logValidator: %v, %v", actualRequest.Source, err)
		}
	case *LogValidatorAssertRequest:
		response.Response, err = s.assert(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run logValidator: %v, %v", actualRequest, err)
		}
	case *LogValidatorResetRequest:
		response.Response, err = s.reset(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run logValidator: %v, %v", actualRequest, err)
		}

	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
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

func (s *logValidatorService) assert(context *Context, request *LogValidatorAssertRequest) (*AssertionInfo, error) {
	var response = &AssertionInfo{}
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

	for _, expectedLogRecords := range request.ExpectedLogRecords {
		logTypeMeta, err := s.getLogTypeMeta(expectedLogRecords, state)
		if err != nil {
			return nil, err
		}
		var logRecordIterator = logTypeMeta.LogRecordIterator()
		logWaitRetryCount := request.LogWaitRetryCount
		logWaitDuration := time.Duration(request.LogWaitTimeMs) * time.Millisecond

		for _, expectedLogRecord := range expectedLogRecords.Records {

			for j := 0; j < logWaitRetryCount; j++ {
				if logRecordIterator.HasNext() {
					break
				}
				s.AddEvent(context, SleepEventType, Pairs("sleepTime", logWaitDuration))
				time.Sleep(logWaitDuration)
			}

			if !logRecordIterator.HasNext() {
				s.AddEvent(context, "Assert", Pairs("actual", nil, "expected", expectedLogRecord, "tag", expectedLogRecords.Tag))
				response.AddFailure(fmt.Sprintf("Missing log record expectedLogRecord :%v", expectedLogRecord))
				return response, nil
			}

			var logRecord = &LogRecord{}
			logRecordIterator.Next(&logRecord)
			_, filename := toolbox.URLSplit(logRecord.URL)
			var actualLogRecord, err = logRecord.AsMap()
			if err != nil {
				return nil, err
			}

			var assertInfo = &AssertionInfo{}
			err = validator.Assert(expectedLogRecord, actualLogRecord, assertInfo, fmt.Sprintf("[%v:%v]", filename, logRecord.Number))
			s.AddEvent(context, "Assert", Pairs("actual", actualLogRecord, "expected", expectedLogRecord, "tag", expectedLogRecords.Tag, "assertInfo", assertInfo))
			response.TestPassed += assertInfo.TestPassed
			if len(assertInfo.TestFailed) > 0 {
				response.TestFailed = append(response.TestFailed, assertInfo.TestFailed...)
			}
			if err != nil {
				return nil, err
			}

		}
	}
	return response, nil
}
func (s *logValidatorService) getLogTypeMeta(expectedLogRecords *ExpectedLogRecord, state data.Map) (*LogTypeMeta, error) {
	var key = logTypeMetaKey(expectedLogRecords.Type)
	s.Mutex().Lock()
	defer s.Mutex().Unlock()
	if !state.Has(key) {
		return nil, fmt.Errorf("Failed to assert, unknown type:%v, please call listen function with requested log type", expectedLogRecords.Type)
	}
	logTypeMeta := state.Get(key).(*LogTypeMeta)
	return logTypeMeta, nil
}

func (s *logValidatorService) loadLogTypeMeta(context *Context, source *url.Resource, logType *LogType) (*LogTypeMeta, error) {
	var logTypesMeta, err = s.readLogFiles(context, source, logType)
	if err != nil {
		return nil, err
	}
	return logTypesMeta[logType.Name], nil
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
	if !has {
		isNewLogFile = true
		logFile = &LogFile{
			Name:            name,
			SkipExpression:  logType.Exclude,
			URL:             candidate.URL(),
			LastModified:    candidate.LastModified(),
			Size:            int(candidate.Size()),
			ProcessingState: &LogProcessingState{},
			Mutex:           &sync.RWMutex{},
			Records:         make([]*LogRecord, 0),
		}
		result.LogFiles[name] = logFile
	}
	s.Mutex().Unlock()

	if !isNewLogFile && (logFile.Size == int(candidate.Size()) && logFile.LastModified.Unix() == candidate.LastModified().Unix()) {
		return result, nil
	}

	reader, err := service.Download(candidate)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var content = string(data)

	var fileOverriden = false
	if len(logFile.Content) > len(content) { //log shrink or rolled over case
		logFile.Reset(candidate)
		logFile.Content = content
		fileOverriden = true
	}

	var contentLength = len(logFile.Content)
	if contentLength > 50 {
		contentLength = 50
	}

	if !fileOverriden && logFile.Size < int(candidate.Size()) && !strings.HasPrefix(content, string(logFile.Content)) {
		if contentLength > len(content) {
			contentLength = len(content)
		}
		logFile.Reset(candidate)
	}
	logFile.Content = content
	logFile.Size = len(data)
	if len(data) > 0 {
		err = logFile.readLogRecords(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *logValidatorService) readLogFiles(context *Context, source *url.Resource, logTypes ...*LogType) (LogTypesMeta, error) {
	var err error
	source, err = context.ExpandResource(source)
	if err != nil {
		return nil, err
	}
	service, err := storage.NewServiceForURL(source.URL, source.Credential)
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

func (s *logValidatorService) listenForChanges(context *Context, request *LogValidatorListenRequest) {
	go func() {
		frequency := time.Duration(request.FrequencyMs) * time.Millisecond
		if frequency == 0 {
			frequency = 250 * time.Millisecond
		}
		for !context.IsClosed() {
			_, err := s.readLogFiles(context, request.Source, request.Types...)
			if err != nil {
				log.Printf("Failed to load log types %v", err)
				break
			}
			time.Sleep(frequency)
		}
	}()
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

	logTypeMetas, err := s.readLogFiles(context, request.Source, request.Types...)
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

	defer s.listenForChanges(context, request)
	return response, nil
}

//NewRequest creates a new request for provided action, (listen, asset,reset)
func (s *logValidatorService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "listen":
		return &LogValidatorListenRequest{}, nil
	case "assert":
		return &LogValidatorAssertRequest{}, nil
	case "reset":
		return &LogValidatorResetRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewLogValidatorService creates a new log validator service.
func NewLogValidatorService() Service {
	var result = &logValidatorService{
		AbstractService: NewAbstractService(LogValidatorServiceID),
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
		return fmt.Errorf("ExpectedLogRecords *%T buy had %T", &LogRecord{}, itemPointer)
	}
	logFile := i.logFiles[i.logFileIndex]
	logRecord := logFile.ShiftLogRecord()
	*logRecordPointer = logRecord
	return nil
}

//LogRecordIterator returns log record iterator
func (m *LogTypeMeta) LogRecordIterator() toolbox.Iterator {

	logFileProvider := func(blacklistedLogFiles ...*LogFile) []*LogFile {

		var blacklisted = make(map[string]bool)
		for _, blacklistedFile := range blacklistedLogFiles {
			blacklisted[blacklistedFile.Name] = true
		}
		var result = make([]*LogFile, 0)
		for _, logFile := range m.LogFiles {
			if _, has := blacklisted[logFile.Name]; has {
				continue
			}

			if logFile.HasPendingLogs() {
				result = append(result, logFile)
			}
		}
		sort.Slice(result, func(i, j int) bool {
			var left = result[i].LastModified
			var right = result[j].LastModified
			if !left.After(*right) && !right.After(*left) {
				return result[i].URL > result[j].URL
			}
			return left.After(*right)
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
