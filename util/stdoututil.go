package util

import (
	"github.com/lunixbochs/vtclean"
	"strings"
	"unicode"
)

const CommandNotFound = "command not found"
const NoSuchFileOrDirectory = "no such file or directory"
const ProgramCanBeFound = "can be found in the following packages"
const ErrorIsNotRecoverable = "Error is not recoverable"
const NotInstalled = "not installed"
const CanNotOpen = "can't open"

var StdErrors = []string{CommandNotFound, NoSuchFileOrDirectory, ProgramCanBeFound}

//CheckNoSuchFileOrDirectory checks for no such file or directory message in the provided stdout.
func CheckNoSuchFileOrDirectory(stdout ...string) bool {
	if len(stdout) == 0 {
		return false
	}
	candidate := strings.ToLower(strings.Join(stdout, "\n"))
	return strings.Contains(candidate, NoSuchFileOrDirectory)
}

//CheckCommandNotFound checks for command not found message in the provided stdout.
func CheckCommandNotFound(stdout ...string) bool {
	if len(stdout) == 0 {
		return false
	}
	candidate := strings.ToLower(strings.Join(stdout, "\n"))
	return strings.Contains(candidate, CommandNotFound) || strings.Contains(candidate, NotInstalled) || strings.Contains(candidate, CanNotOpen)
}

//ExtractColumn extract a column from the line for provided index
func ExtractColumn(line string, columnIndex int) (string, bool) {
	var columns, has = ExtractColumns(line)
	if !has {
		return "", has
	}
	if columnIndex < len(columns) {
		return columns[columnIndex], true
	}
	return "", false
}

//ExtractColumns extract all column from the line
func ExtractColumns(line string) ([]string, bool) {
	if line == "" {
		return []string{}, false
	}
	var index = -1
	var expectColumn = true
	var result = make([]string, 0)
	for _, r := range line {
		if unicode.IsSpace(r) {
			expectColumn = true
			continue
		}
		if expectColumn {
			index++
			result = append(result, "")
			expectColumn = false
		}

		result[index] += string(r)
	}

	for i, value := range result {
		result[i] = vtclean.Clean(value, false)
	}
	return result, true
}

func EscapeStdout(stdout string) string {
	return vtclean.Clean(stdout, true)
}

func EscapedContains(stdout, match string) bool {
	return strings.Contains(stdout, match) || strings.Contains(EscapeStdout(stdout), match)
}
