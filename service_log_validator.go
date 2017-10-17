package endly

import (
	"fmt"
	"github.com/viant/toolbox/storage"
	"time"
	"strings"
	"regexp"
	"github.com/viant/toolbox"
	"sort"
	"io"
	"io/ioutil"
)

const LogValidatorServiceId = "validator/log"

type logValidatorService struct {
	*AbstractService
}

type LogType struct {
	Name   string
	Format string
	Mask   string
}

type LogValidatorListenRequest struct {
	Source *Resource
	Types  []*LogType
}

type LogProcessingState struct {
	Line     int
	Position int
}

type LogFile struct {
	URL             string
	Name            string
	ProcessingState *LogProcessingState
	LastModified    *time.Time
	Size            int
}

func (f *LogFile) HasPendingLogs() bool {
	if f.ProcessingState == nil {
		f.ProcessingState = &LogProcessingState{}
	}
	return f.Size != f.ProcessingState.Position
}

func (f *LogFile) processLog(reader io.Reader, handler func(url string, lineIndex int, line string) (bool, error)) (bool, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return false, err
	}
	if f.ProcessingState.Position > 0 {
		data = data[f.ProcessingState.Position:]
	}
	var lineIndex = f.ProcessingState.Line
	var line = ""
	var dataProcessed = 0
	for i := 0; i < len(data); i++ {
		dataProcessed++;
		aChar := string(data[i])
		if aChar != "\n" && aChar != "\r" {
			line += aChar
			continue
		}
		lineIndex++
		line = strings.Trim(line, " \r\t")
		next, err := handler(f.URL, lineIndex, line)
		line = ""
		if err != nil {
			return false, err
		}
		if ! next {
			return next, nil
		}
		f.ProcessingState.Line = lineIndex
		f.ProcessingState.Position += dataProcessed
		dataProcessed = 0
	}

	return true, nil
}

type LogTypeMeta struct {
	Source  *Resource
	LogType *LogType
	Info    map[string]*LogFile
}

func (m *LogTypeMeta) Range(initial *LogTypeMeta, handler func(url string, index int, line string) (bool, error)) error {
	var candidates = make([]*LogFile, 0)

	for name, info := range m.Info {
		var initialInfo, has = initial.Info[name]
		if has {
			if info.Size == initialInfo.Size && ! initialInfo.HasPendingLogs() { //size has not change, and all data has been processed, fully processed
				continue
			}
			info.ProcessingState = initialInfo.ProcessingState
		}
		candidates = append(candidates, info)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].LastModified.Unix() > candidates[j].LastModified.Unix()
	})

	var service, err = storage.NewServiceForURL(m.Source.URL, m.Source.Credential)
	if err != nil {
		return err
	}

	for i, candidate := range candidates {
		object, err := service.StorageObject(candidate.URL)
		if err != nil {
			return err
		}
		reader, err := service.Download(object)
		next, err := candidates[i].processLog(reader, handler)
		if err != nil {
			return err
		}
		if ! next {
			return nil
		}
	}
	return nil
}

func NewLogTypeMeta(source *Resource, logType *LogType) *LogTypeMeta {
	return &LogTypeMeta{
		Source:  source,
		LogType: logType,
		Info:    make(map[string]*LogFile),
	}
}

type LogTypesMeta map[string]*LogTypeMeta

type LogValidatorListenResponse struct {
	Meta LogTypesMeta
}

type LogValidatorAssertRequest struct {
	Type string
	Data []map[string]interface{}
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
			response.Error = fmt.Sprintf("Failed to run logValidator: %v, %v", actualRequest.Type, err)
		}
	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *logValidatorService) assert(context *Context, request *LogValidatorAssertRequest) (*ValidatorAssertionInfo, error) {
	var key = LogTypeMetaKey(request.Type)
	var state = s.State()
	if ! state.Has(key) {
		return nil, fmt.Errorf("Failed to assert, unknown type:%v, please call listen function with requested log type", request.Type)
	}
	initialLogTypeMeta := state.Get(key).(*LogTypeMeta)
	logTypeMeta, err := s.loadLogTypeMeta(context, initialLogTypeMeta.Source, initialLogTypeMeta.LogType)
	defer state.Put(key, logTypeMeta)
	if err != nil {
		return nil, err
	}
	var expectedIndex = 0
	var response = &ValidatorAssertionInfo{}
	startEvent := s.Begin(context, request, Pairs("logType", request.Type, "expected", request.Data), Info)
	defer s.End(context)(startEvent, Pairs("response", response))
	err = logTypeMeta.Range(initialLogTypeMeta, func(url string, lineIndex int, line string) (bool, error) {
		if expectedIndex >= len(request.Data) {
			return false, nil
		}
		_, filename := toolbox.URLSplit(url)
		if logTypeMeta.LogType.Format == "json" {
			var actual = make(map[string]interface{})
			err = toolbox.NewJSONDecoderFactory().Create(strings.NewReader(line)).Decode(&actual)
			if err != nil {
				return false, err
			}
			var expected = request.Data[expectedIndex]
			expectedIndex++
			validator := &Validator{
				SkipFields: make(map[string]bool),
			}
			err := validator.Assert(expected, actual, response, fmt.Sprintf("[%v:%v]", filename, lineIndex))
			if err != nil {
				return false, err
			}

		} else {
			return false, fmt.Errorf("Unsupported format: %v", logTypeMeta.LogType.Format)
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (s *logValidatorService) loadLogTypeMeta(context *Context, source *Resource, logType *LogType) (*LogTypeMeta, error) {
	var logTypesMeta, err = s.loadLogTypesMeta(context, source, logType)
	if err != nil {
		return nil, err
	}
	return logTypesMeta[logType.Name], nil
}

func (s *logValidatorService) loadLogTypesMeta(context *Context, source *Resource, logTypes ... *LogType) (LogTypesMeta, error) {
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
			maskExpression, err := regexp.Compile(mask)
			if err != nil {
				return nil, err
			}
			_, name := toolbox.URLSplit(candidate.URL())
			if maskExpression.MatchString(name) {
				if _, has := response[logType.Name]; !has {
					response[logType.Name] = NewLogTypeMeta(source, logType)
				}
				logTypeMeta := response[logType.Name]
				logInfo := &LogFile{
					Name:         name,
					URL:          candidate.URL(),
					LastModified: candidate.LastModified(),
					Size:         int(candidate.Size()),
					ProcessingState: &LogProcessingState{

					},
				}
				logTypeMeta.Info[logInfo.Name] = logInfo
			}
		}
	}
	return response, nil
}

func (s *logValidatorService) listen(context *Context, request *LogValidatorListenRequest) (*LogValidatorListenResponse, error) {
	var source, err = context.ExpandResource(request.Source)
	if err != nil {
		return nil, err
	}
	loadLogTypeMeta, err := s.loadLogTypesMeta(context, request.Source, request.Types...)
	if err != nil {
		return nil, err
	}
	for _, logType := range request.Types {

		logMeta, ok := loadLogTypeMeta[logType.Name]
		if ! ok {
			logMeta = NewLogTypeMeta(source, logType)
			loadLogTypeMeta[logType.Name] = logMeta
		}
		s.state.Put(LogTypeMetaKey(logType.Name), logMeta)
	}
	response := &LogValidatorListenResponse{
		Meta: loadLogTypeMeta,
	}
	return response, nil
}

func (s *logValidatorService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "listen":
		return &LogValidatorListenRequest{}, nil
	case "assert":
		return &LogValidatorAssertRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func NewLogValidatorService() Service {
	var result = &logValidatorService{
		AbstractService: NewAbstractService(LogValidatorServiceId),
	}
	result.AbstractService.Service = result
	return result
}

func LogTypeMetaKey(name string) string {
	return fmt.Sprintf("meta_%v", name)
}
