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

func ExtractColumn(stdout string, columnIndex int) (string, bool) {
	if stdout == "" {
		return "", false
	}
	var result = ""
	var index = -1
	var expectColumn = true

	for i := 0; i < len(stdout); i++ {
		var aChar = string(stdout[i : i+1])
		if aChar == " " || aChar == "\t" {
			expectColumn = true
			continue
		}
		if expectColumn {
			index++
			expectColumn = false
		}
		if columnIndex == index {
			result += aChar
		}
		if index > columnIndex {
			break
		}
	}
	if result == "" {
		return "", false
	}
	return result, true
}


