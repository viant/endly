package endly

import (
	"fmt"
	"strings"
)

//CommandLog represents an executed command with Stdin, Stdout or Error
type CommandLog struct {
	Stdin  string
	Stdout string
	Error  string
}

//CommandResponse represents a command response with logged commands.
type CommandResponse struct {
	Session   string
	Commands  []*CommandLog
	Extracted map[string]string
	Error     string
}

//Add appends provided log into commands slice.
func (i *CommandResponse) Add(log *CommandLog) {
	if len(i.Commands) == 0 {
		i.Commands = make([]*CommandLog, 0)
	}
	i.Commands = append(i.Commands, log)
}

//Stdout returns stdout for provided index, or all concatenated otherwise
func (i *CommandResponse) Stdout(indexes ...int) string {
	if len(indexes) == 0 {
		var result = make([]string, len(i.Commands))
		for j, stream := range i.Commands {
			result[j] = stream.Stdout
		}
		return strings.Join(result, "\r\n")
	}
	var result = make([]string, len(indexes))
	for _, index := range indexes {
		if index < len(i.Commands) {
			result = append(result, i.Commands[index].Stdout)
		}
	}
	return strings.Join(result, "\r\n")
}

//NewCommandResponse creates a new CommandResponse
func NewCommandResponse(session string) *CommandResponse {
	return &CommandResponse{
		Session:   session,
		Commands:  make([]*CommandLog, 0),
		Extracted: make(map[string]string),
	}
}

//NewCommandLog creates a new command log
func NewCommandLog(stdin, stdout string, err error) *CommandLog {
	result := &CommandLog{
		Stdin: stdin,
	}
	if err != nil {
		result.Error = fmt.Sprintf("%v", err)
	} else {
		result.Stdout = stdout
	}
	return result
}
