package endly

import (
	"github.com/lunixbochs/vtclean"
	"strings"
)

const commandNotFound = "command not found"
const noSuchFileOrDirectory = "no such file or directory"
const programCanBeFound = "can be found in the following packages"

//CheckNoSuchFileOrDirectory checks for no such file or directory message in the provided stdout.
func CheckNoSuchFileOrDirectory(stdout ...string) bool {
	if len(stdout) == 0 {
		return false
	}
	candidate := strings.ToLower(strings.Join(stdout, "\n"))
	return strings.Contains(candidate, noSuchFileOrDirectory)
}

//CheckCommandNotFound checks for command not found message in the provided stdout.
func CheckCommandNotFound(stdout ...string) bool {
	if len(stdout) == 0 {
		return false
	}
	candidate := strings.ToLower(strings.Join(stdout, "\n"))
	return strings.Contains(candidate, commandNotFound)
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
	line = vtclean.Clean(line, false)
	var index = -1
	var expectColumn = true
	var result = make([]string, 0)

	for i := 0; i < len(line); i++ {
		var aChar = string(line[i : i+1])
		if aChar == " " || aChar == "\t" {
			expectColumn = true
			continue
		}
		if expectColumn {
			index++
			result = append(result, "")
			expectColumn = false
		}
		result[index] += aChar
	}
	return result, true
}

func escapeStdout(stdout string) string {
	return vtclean.Clean(stdout, true)
}
