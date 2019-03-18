package docker

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/viant/endly"
	"io"
	"strings"
)

//DataError represent data error
type DataError struct {
	Message string `json:"message"`
}

type DataAux struct {
	Tag    string
	Digest string
	Size   int
}

//DataStream represents a data strem
type DataStream struct {
	Stream      string     `json:"stream"`
	Progress    string     `json:"progress"`
	Status      string     `json:"status"`
	Id          string     `json:"id"`
	ErrorDetail *DataError `json:"errorDetail"`
	Aux         *DataAux   `json:"aux"`
	Error       string     `json:"error"`
}

func (s *DataStream) Data() string {
	result := ""

	if s.ErrorDetail != nil && s.ErrorDetail.Message != "" {
		return s.ErrorDetail.Message
	}

	if strings.TrimSpace(s.Stream) != "" {
		return s.Stream
	}
	if s.Status == "" && s.Progress == "" {
		return s.Stream
	}
	status := s.Status
	hasExtendedStatus := len(status) >= 30
	if hasExtendedStatus {
		result = s.Status + "\n"
		status = strings.Repeat(" ", 29)
	}

	progress := strings.Replace(s.Progress, "\n", "", 1)
	result += fmt.Sprintf("%-30s%v", status, progress)
	if strings.Count(result, "\n") == 0 {
		result = "\r" + result
	}
	if hasExtendedStatus {
		result += "\n"
	}
	return result
}

func readStream(context *endly.Context, tag string, reader io.Reader, stdout *[]string, callback func(stream *DataStream)) error {
	scanner := bufio.NewScanner(reader)
	stream := &DataStream{}
	var status = ""
	for scanner.Scan() {
		rawData := scanner.Bytes()
		err := json.Unmarshal(rawData, &stream)
		if err != nil {
			continue
		}
		data := stream.Data()
		if data == "" {
			data = string(rawData)
		}
		if status != stream.Status ||
			stream.Status == "" && strings.Count(data, "\n") == 0 {
			data += "\n"
		}
		status = stream.Status

		if callback != nil {
			callback(stream)
		}
		*stdout = append(*stdout, data)
		context.Publish(NewStdoutEvent(tag, data))
	}
	return nil
}
