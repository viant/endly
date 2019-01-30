package cloudfunctions

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"io/ioutil"
	"strings"
)

const defaultRegion = "us-central1"
const parentLocationTemplate = "projects/${gc.projectID}/locations/${gc.region}"


func initFullyQualifiedName(name string) string {
	if  strings.HasPrefix(name, "projects/") {
		return name
	}
	return parentLocationTemplate + "/functions/" + name
}

func initRegion(region string) string {
	if region != "" {
		return region
	}
	return defaultRegion
}


func getIgnoreList(service storage.Service, URL string) []string {
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
	return strings.Split(toolbox.AsString(content),",")
}