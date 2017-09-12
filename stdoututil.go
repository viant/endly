package endly

import (
	"strings"
)

const commandNotFound = "command not found"
const noSuchFileOrDirectory = "no such file or directory"

func CheckNoSuchFileOrDirectory(stdout ...string) bool {
	if len(stdout) == 0 {
		return false
	}
	candidate := strings.ToLower(strings.Join(stdout, "\n"))
	return strings.Contains(candidate, noSuchFileOrDirectory)
}

func CheckCommandNotFound(stdout ...string) bool {
	if len(stdout) == 0 {
		return false
	}
	candidate := strings.ToLower(strings.Join(stdout, "\n"))
	return strings.Contains(candidate, commandNotFound)
}

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

func ExtractColumns(line string) ([]string, bool) {
	if line == "" {
		return []string{}, false
	}
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
