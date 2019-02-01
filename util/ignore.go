package util

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"io/ioutil"
	"strings"
)

//GetIgnoreList returns ignore list
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
