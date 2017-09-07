package endly

import (
	"github.com/viant/toolbox"
	"strings"
)

func CheckNoSuchFileOrDirectory(stdout []string) bool {
	if len(stdout) == 0 {
		return false
	}
	candidate := strings.ToLower(strings.Join(stdout, "\n"))
	return strings.Contains(candidate, "no such file or directory")
}

func CheckCommandNotFound(stdout []string) bool {
	if len(stdout) == 0 {
		return false
	}
	candidate := strings.ToLower(strings.Join(stdout, "\n"))
	return strings.Contains(candidate, "command not found")
}

func ExtractColumnAsInt(stdout string, columnIndex int) (int, bool) {
	if stdout == "" {
		return 0, false
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
		return 0, false
	}
	return toolbox.AsInt(result), true
}
