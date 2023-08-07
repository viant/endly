package util

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"io/ioutil"
	"strings"
)

// GetIgnoreList returns ignore list
func GetIgnoreList(service storage.Service, URL string) []string {
	var list = make([]string, 0)
	reader, err := service.DownloadWithURL(URL)
	if err != nil {
		return list
	}
	defer reader.Close()
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return list
	}
	for _, item := range strings.Split(toolbox.AsString(content), "\n") {
		if strings.HasPrefix(item, "#") {
			continue
		}
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		list = append(list, strings.TrimSpace(item))
	}
	return list
}

func ShouldIgnoreLocation(location string, ignoreList []string) bool {

	filename := location
	if index := strings.LastIndex(location, "/"); index != -1 {
		filename = string(location[index+1:])
	}

	for _, expr := range ignoreList {

		if filename == expr {
			return true
		} else if strings.HasPrefix(expr, "/") {
			prefix := expr[1:]
			if strings.HasPrefix(location, prefix) && prefix != location {
				return true
			}
		} else if strings.HasSuffix(expr, "/**") {
			index := strings.LastIndex(expr, "/**")
			prefix := string(expr[0:index])
			if strings.HasPrefix(location, prefix) {
				return true
			}
		} else if strings.HasSuffix(expr, "/") {
			index := strings.LastIndex(expr, "/")
			prefix := string(expr[0:index])
			if strings.HasPrefix(location, prefix) {
				return true
			}
		} else if strings.HasPrefix(expr, "**/") {
			index := strings.Index(expr, "**/")
			suffix := string(expr[index+3:])
			if strings.HasSuffix(location, suffix) {
				return true
			}
		} else if strings.HasSuffix(expr, "*") {
			index := strings.Index(expr, "*")
			prefix := expr[:index]
			if strings.HasPrefix(filename, prefix) {
				return true
			}

		} else if strings.HasPrefix(expr, "*") {
			index := strings.Index(expr, "*")
			suffix := expr[index+1:]
			if strings.HasSuffix(filename, suffix) {
				return true
			}

		} else if strings.Contains(expr, "*") {
			index := strings.Index(expr, "*")
			prefix := expr[:index]
			suffix := expr[index+1:]
			if strings.HasPrefix(filename, prefix) && strings.HasSuffix(filename, suffix) {
				return true
			}

		}
	}
	return false
}
